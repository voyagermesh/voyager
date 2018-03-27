package controller

import (
	"bytes"
	"strings"
	"time"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	voyagerinformers "github.com/appscode/voyager/client/informers/externalversions"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Options struct {
	CloudProvider  string
	IngressRef     core.ObjectReference
	ConfigDir      string
	CertDir        string
	CmdFile        string
	QPS            float32
	Burst          int
	MaxNumRequeues int
	NumThreads     int
	ResyncPeriod   time.Duration
}

func (opts Options) UsesEngress() bool {
	return opts.IngressRef.APIVersion == api.SchemeGroupVersion.String()
}

type Controller struct {
	k8sClient              kubernetes.Interface
	VoyagerClient          cs.Interface
	options                Options
	kubeInformerFactory    informers.SharedInformerFactory
	voyagerInformerFactory voyagerinformers.SharedInformerFactory
	recorder               record.EventRecorder

	store *certificate.CertStore

	cfgWriter  *ioutilz.AtomicWriter
	certWriter *ioutilz.AtomicWriter

	cfgQueue    *queue.Worker
	cfgInformer cache.SharedIndexInformer

	secretQueue    *queue.Worker
	secretInformer cache.SharedIndexInformer

	ingQueue    *queue.Worker
	ingInformer cache.SharedIndexInformer

	engQueue    *queue.Worker
	engInformer cache.SharedIndexInformer

	crtQueue    *queue.Worker
	crtInformer cache.SharedIndexInformer
}

func New(client kubernetes.Interface, voyagerClient cs.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:              client,
		kubeInformerFactory:    informers.NewFilteredSharedInformerFactory(client, opt.ResyncPeriod, opt.IngressRef.Namespace, nil),
		VoyagerClient:          voyagerClient,
		voyagerInformerFactory: voyagerinformers.NewFilteredSharedInformerFactory(voyagerClient, opt.ResyncPeriod, opt.IngressRef.Namespace, nil),
		options:                opt,
		recorder:               eventer.NewEventRecorder(client, "haproxy-controller"),
	}
}

func (c *Controller) getIngressWorker() *queue.Worker {
	if c.options.UsesEngress() {
		return c.engQueue
	}
	return c.ingQueue
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
	c.initConfigMapWatcher()
	c.initSecretWatcher()
	c.initCertificateCRDWatcher()
	c.store, err = certificate.NewCertStore(c.k8sClient, c.VoyagerClient)
	if err != nil {
		return
	}
	c.cfgWriter, err = ioutilz.NewAtomicWriter(strings.TrimSuffix(c.options.ConfigDir, "/"))
	if err != nil {
		return
	}
	c.certWriter, err = ioutilz.NewAtomicWriter(strings.TrimSuffix(c.options.CertDir, "/"))
	if err != nil {
		return
	}

	var ing *api.Ingress
	ing, err = c.initIngressIndexer()
	if err != nil {
		return
	}
	err = c.initConfigCache(ing)
	if err != nil {
		return
	}
	err = c.initTLSCache(ing)
	if err != nil {
		return
	}
	err = c.mountIngress(ing)
	return
}

func (c *Controller) initIngressIndexer() (*api.Ingress, error) {
	if c.options.UsesEngress() {
		obj, err := c.VoyagerClient.VoyagerV1beta1().Ingresses(c.options.IngressRef.Namespace).Get(c.options.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		obj.Migrate()
		err = obj.IsValid(c.options.CloudProvider)
		if err != nil {
			return nil, err
		}
		c.engInformer.GetIndexer().Add(obj)
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
	c.ingInformer.GetIndexer().Add(obj)
	return ingress, nil
}

func (c *Controller) initConfigCache(ing *api.Ingress) error {
	cm, err := c.k8sClient.CoreV1().ConfigMaps(c.options.IngressRef.Namespace).Get(api.VoyagerPrefix+c.options.IngressRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return c.cfgInformer.GetIndexer().Add(cm)
}

func (c *Controller) initTLSCache(ing *api.Ingress) error {
	for _, tls := range ing.Spec.TLS {
		if strings.EqualFold(tls.Ref.Kind, api.ResourceKindCertificate) {
			crd, err := c.VoyagerClient.VoyagerV1beta1().Certificates(c.options.IngressRef.Namespace).Get(tls.Ref.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			err = c.crtInformer.GetIndexer().Add(crd)
			if err != nil {
				return err
			}
		} else {
			sc, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(tls.Ref.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			err = c.secretInformer.GetIndexer().Add(sc)
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
		err = c.secretInformer.GetIndexer().Add(stls)
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
				err = c.secretInformer.GetIndexer().Add(stls)
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

func runCmd() error {
	glog.Info("Running haproxy start/reload...")
	if err := startOrReloadHaproxy(); err != nil {
		return err
	}
	incReloadCounter()
	return nil
}

func (c *Controller) Run(stopCh chan struct{}) {
	defer runtime.HandleCrash()

	glog.Info("Starting haproxy-controller")
	c.kubeInformerFactory.Start(stopCh)
	c.voyagerInformerFactory.Start(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, v := range c.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(errors.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	for _, v := range c.voyagerInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(errors.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	c.cfgQueue.Run(stopCh)
	c.secretQueue.Run(stopCh)
	c.getIngressWorker().Run(stopCh)
	c.crtQueue.Run(stopCh)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			if _, err := checkHAProxyDaemon(); err != nil {
				startHAProxy()
			}
		}
	}()

	<-stopCh
	glog.Info("Stopping haproxy-controller")
}
