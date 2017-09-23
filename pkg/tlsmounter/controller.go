package tlsmounter

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/workqueue"
)

type Options struct {
	CloudProvider  string
	IngressRef     apiv1.ObjectReference
	MountPath      string
	CmdFile        string
	ResyncPeriod   time.Duration
	MaxNumRequeues int
}

func (opts Options) UsesEngress() bool {
	return opts.IngressRef.Kind == api.SchemeGroupVersion.String()
}

type Controller struct {
	k8sClient     kubernetes.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	options       Options
	recorder      record.EventRecorder

	store   *certificate.CertStore
	Ingress *api.Ingress

	lock        sync.Mutex
	writer      *ioutil.AtomicWriter
	projections map[string]ioutil.FileProjection

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
	switch c.options.IngressRef.APIVersion {
	case api.SchemeGroupVersion.String():
		c.Ingress, err = c.VoyagerClient.Ingresses(c.options.IngressRef.Namespace).Get(c.options.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return
		}
		err = c.Ingress.IsValid(c.options.CloudProvider)
		if err != nil {
			return
		}
		c.initIngressCRDWatcher()
	case extensions.SchemeGroupVersion.String():
		var ing *extensions.Ingress
		ing, err = c.k8sClient.ExtensionsV1beta1().Ingresses(c.options.IngressRef.Namespace).Get(c.options.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return
		}
		c.Ingress, err = api.NewEngressFromIngress(ing)
		if err != nil {
			return
		}
		err = c.Ingress.IsValid(c.options.CloudProvider)
		if err != nil {
			return
		}
		c.initIngressWatcher()
	default:
		err = errors.New("ingress API Schema unrecognized")
		return
	}

	c.initSecretWatcher()
	c.initCertificateCRDWatcher()
	c.store, err = certificate.NewCertStore(c.k8sClient, c.VoyagerClient)
	if err != nil {
		return
	}

	projections := make(map[string]ioutil.FileProjection)
	for _, tls := range c.Ingress.Spec.TLS {
		if tls.SecretRef.Kind == "Certificate" {
			var crd *api.Certificate
			if crd, err = c.VoyagerClient.Certificates(c.options.IngressRef.Namespace).Get(tls.SecretRef.Name, metav1.GetOptions{}); err == nil {
				var crt *x509.Certificate
				var key *rsa.PrivateKey
				crt, key, err = c.store.Get(crd)
				if err != nil {
					return
				}
				projections[c.store.Name(crd)+".pem"] = ioutil.FileProjection{Mode: 0777, Data: certificateToPEMData(crt, key)}
			}
		} else {
			var sc *apiv1.Secret
			if sc, err = c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(tls.SecretRef.Name, metav1.GetOptions{}); err == nil {
				var data []byte
				data, err = secretToPEMData(sc)
				if err != nil {
					return
				}
				projections[tls.SecretRef.Name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: data}
			}
		}
	}
	c.projections = projections

	c.writer, err = ioutil.NewAtomicWriter(strings.TrimSuffix(c.options.MountPath, "/"))
	if err != nil {
		return
	}
	err = c.writer.Write(c.projections)
	return
}

func secretToPEMData(s *apiv1.Secret) ([]byte, error) {
	var buf bytes.Buffer
	if data, found := s.Data[apiv1.TLSCertKey]; found {
		buf.Write(data)
	} else {
		return nil, fmt.Errorf("secret %s@%s is missing required key %s", s.Name, s.Namespace, apiv1.TLSCertKey)
	}
	buf.WriteRune('\n')
	if data, found := s.Data[apiv1.TLSPrivateKeyKey]; found {
		buf.Write(data)
	} else {
		return nil, fmt.Errorf("secret %s@%s is missing required key %s", s.Name, s.Namespace, apiv1.TLSPrivateKeyKey)
	}
	return buf.Bytes(), nil
}

func certificateToPEMData(crt *x509.Certificate, key *rsa.PrivateKey) []byte {
	var buf bytes.Buffer
	buf.Write(cert.EncodeCertPEM(crt))
	buf.WriteRune('\n')
	buf.Write(cert.EncodePrivateKeyPEM(key))
	return buf.Bytes()
}

var updateReceived, mountPerformed uint64

func incUpdateReceivedCounter() {
	atomic.AddUint64(&updateReceived, 1)
	log.Infoln("Update Received:", atomic.LoadUint64(&updateReceived))
}

func incMountCounter() {
	atomic.AddUint64(&mountPerformed, 1)
	log.Infoln("Mount Performed:", atomic.LoadUint64(&mountPerformed))
}

func runCmd(path string) error {
	log.Infoln("calling boot file to execute")
	output, err := exec.Command("sh", "-c", path).CombinedOutput()
	msg := fmt.Sprintf("%v", string(output))
	log.Infoln("Output:\n", msg)
	if err != nil {
		log.Errorln("failed to run cmd")
		return fmt.Errorf("error restarting %v: %v", msg, err)
	}
	log.Infoln("boot file executed")
	return nil
}

func (c *Controller) initSecretWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Watch(metav1.ListOptions{})
		},
	}

	// create the workqueue
	c.sQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "secret")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	c.sIndexer, c.sInformer = cache.NewIndexerInformer(lw, &apiv1.Secret{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.sQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.sQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.sQueue.Add(key)
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) initIngressWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).List(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).Watch(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
	}

	// create the workqueue
	c.ingQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	c.ingIndexer, c.ingInformer = cache.NewIndexerInformer(lw, &apiv1.Secret{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.ingQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.ingQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.ingQueue.Add(key)
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) initIngressCRDWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.VoyagerClient.Ingresses(c.Ingress.Namespace).List(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.VoyagerClient.Ingresses(c.Ingress.Namespace).Watch(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
	}

	// create the workqueue
	c.engQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress-crd")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	c.engIndexer, c.engInformer = cache.NewIndexerInformer(lw, &apiv1.Secret{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.engQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.engQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.engQueue.Add(key)
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) initCertificateCRDWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.VoyagerClient.Certificates(c.options.IngressRef.Namespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.VoyagerClient.Certificates(c.options.IngressRef.Namespace).Watch(metav1.ListOptions{})
		},
	}

	// create the workqueue
	c.cQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "certificate-crd")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	c.cIndexer, c.cInformer = cache.NewIndexerInformer(lw, &apiv1.Secret{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.cQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.cQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.cQueue.Add(key)
			}
		},
	}, cache.Indexers{})
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
