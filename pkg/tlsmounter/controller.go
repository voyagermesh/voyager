package tlsmounter

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Options struct {
	CloudProvider  string
	IngressRef     apiv1.ObjectReference
	MountPath      string
	CmdFile        string
	QPS            float32
	Burst          int
	ResyncPeriod   time.Duration
	MaxNumRequeues int
}

func (opts Options) UsesEngress() bool {
	return opts.IngressRef.APIVersion == api.SchemeGroupVersion.String()
}

type Controller struct {
	k8sClient     kubernetes.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	options       Options
	recorder      record.EventRecorder

	store *certificate.CertStore

	writer *ioutilz.AtomicWriter

	sQueue    workqueue.RateLimitingInterface
	sIndexer  cache.Indexer
	sInformer cache.Controller

	ingQueue    workqueue.RateLimitingInterface
	ingIndexer  cache.Indexer
	ingInformer cache.Controller

	engQueue    workqueue.RateLimitingInterface
	engIndexer  cache.Indexer
	engInformer cache.Controller

	cQueue    workqueue.RateLimitingInterface
	cIndexer  cache.Indexer
	cInformer cache.Controller
}

func New(client kubernetes.Interface, voyagerClient acs.VoyagerV1beta1Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:     client,
		VoyagerClient: voyagerClient,
		options:       opt,
		recorder:      eventer.NewEventRecorder(client, "tls-mounter"),
	}
}

func (c *Controller) Setup() (err error) {
	if c.options.IngressRef.APIVersion == "" {
		err = errors.New("ingress api version not found")
		return
	}
	if c.options.IngressRef.Name == "" {
		err = errors.New("ingress name not found")
		return
	}

	if c.options.IngressRef.APIVersion == api.SchemeGroupVersion.String() {
		c.initIngressCRDWatcher()
	} else {
		c.initIngressWatcher()
	}
	c.initSecretWatcher()
	c.initCertificateCRDWatcher()
	c.store, err = certificate.NewCertStore(c.k8sClient, c.VoyagerClient)
	if err != nil {
		return
	}
	c.writer, err = ioutilz.NewAtomicWriter(strings.TrimSuffix(c.options.MountPath, "/"))
	if err != nil {
		return
	}

	var ing *api.Ingress
	ing, err = c.initIngressIndexer()
	if err != nil {
		return
	}
	err = c.initTLSCache(ing)
	if err != nil {
		return
	}
	// Do not run cmd in initOnly as it will restart the HAProxy
	// But the config map is not still mounted.
	err = c.mountIngress(ing, false)
	return
}

func (c *Controller) initIngressIndexer() (*api.Ingress, error) {
	if c.options.UsesEngress() {
		obj, err := c.VoyagerClient.Ingresses(c.options.IngressRef.Namespace).Get(c.options.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		obj.Migrate()
		err = obj.IsValid(c.options.CloudProvider)
		if err != nil {
			return nil, err
		}
		c.engIndexer.Add(obj)
		return obj, nil
	}

	obj, err := c.k8sClient.ExtensionsV1beta1().Ingresses(c.options.IngressRef.Namespace).Get(c.options.IngressRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ingress, err := api.NewEngressFromIngress(obj)
	if err != nil {
		return nil, err
	}
	err = ingress.IsValid(c.options.CloudProvider)
	if err != nil {
		return nil, err
	}
	c.ingIndexer.Add(obj)
	return ingress, nil
}

func (c *Controller) initTLSCache(ing *api.Ingress) error {
	for _, tls := range ing.Spec.TLS {
		if strings.EqualFold(tls.Ref.Kind, api.ResourceKindCertificate) {
			crd, err := c.VoyagerClient.Certificates(c.options.IngressRef.Namespace).Get(tls.Ref.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			err = c.cIndexer.Add(crd)
			if err != nil {
				return err
			}
		} else {
			sc, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(tls.Ref.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			err = c.sIndexer.Add(sc)
			if err != nil {
				return err
			}
		}
	}

	if name := ing.AuthTLSSecret(); name != "" {
		stls, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		err = c.sIndexer.Add(stls)
		if err != nil {
			return err
		}
	} else {
		for _, fr := range ing.Spec.FrontendRules {
			if fr.Auth != nil && fr.Auth.TLS != nil {
				stls, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(fr.Auth.TLS.SecretName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				err = c.sIndexer.Add(stls)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func certificateToPEMData(crt, key []byte) []byte {
	var buf bytes.Buffer
	buf.Write(bytes.TrimSpace(crt))
	buf.WriteRune('\n')
	buf.Write(bytes.TrimSpace(key))
	return buf.Bytes()
}

var mountPerformed uint64

func incMountCounter() {
	atomic.AddUint64(&mountPerformed, 1)
	log.Infoln("Mount Performed:", atomic.LoadUint64(&mountPerformed))
}

func runCmd(path string) error {
	output, err := exec.Command("sh", "-c", path).CombinedOutput()
	msg := fmt.Sprintf("%v", string(output))
	if err != nil {
		return fmt.Errorf("error restarting %v: %v", msg, err)
	}
	incMountCounter()
	return nil
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.sQueue.ShutDown()
	if c.options.UsesEngress() {
		defer c.engQueue.ShutDown()
	} else {
		defer c.ingQueue.ShutDown()
	}
	defer c.cQueue.ShutDown()
	glog.Info("Starting tls-mounter")

	go c.sInformer.Run(stopCh)
	if c.options.UsesEngress() {
		go c.engInformer.Run(stopCh)
	} else {
		go c.ingInformer.Run(stopCh)
	}
	go c.cInformer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.sInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if c.options.UsesEngress() {
		if !cache.WaitForCacheSync(stopCh, c.engInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	} else {
		if !cache.WaitForCacheSync(stopCh, c.ingInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	if !cache.WaitForCacheSync(stopCh, c.cInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runSecretWatcher, time.Second, stopCh)
		if c.options.UsesEngress() {
			go wait.Until(c.runIngressCRDWatcher, time.Second, stopCh)
		} else {
			go wait.Until(c.runIngressWatcher, time.Second, stopCh)
		}
		go wait.Until(c.runCertificateWatcher, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping tls-mounter")
}
