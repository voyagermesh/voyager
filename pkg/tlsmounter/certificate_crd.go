package tlsmounter

import (
	"strings"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

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
	c.cIndexer, c.cInformer = cache.NewIndexerInformer(lw, &api.Certificate{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(r) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						c.cQueue.Add(key)
					}
				}
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			if r, ok := new.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(r) {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						c.cQueue.Add(key)
					}
				}
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

func (c *Controller) isCertificateUsedInIngress(s *api.Certificate) bool {
	if s.Namespace != c.options.IngressRef.Namespace {
		return false
	}
	r, err := c.getIngress()
	if err != nil {
		return false
	}
	for _, tls := range r.Spec.TLS {
		if s.Name == tls.Ref.Name && strings.EqualFold(tls.Ref.Kind, api.ResourceKindCertificate) {
			return true
		}
	}
	return false
}

func (c *Controller) runCertificateWatcher() {
	for c.processNextCertificate() {
	}
}

func (c *Controller) processNextCertificate() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.cQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.cQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.syncCertificate(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.cQueue.Forget(key)
		return true
	}
	log.Errorln("Failed to process Certificate %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.cQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.cQueue.AddRateLimited(key)
		return true
	}

	c.cQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncCertificate(key string) error {
	key, err := cache.MetaNamespaceKeyFunc(cache.ExplicitKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name))
	if err != nil {
		return err
	}
	if c.options.UsesEngress() {
		c.engQueue.Add(key)
	} else {
		c.ingQueue.Add(key)
	}
	return nil
}

func (c *Controller) getCertificate(name string) (*api.Certificate, error) {
	obj, exists, err := c.cIndexer.GetByKey(c.options.IngressRef.Namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(apiv1.Resource("secret"), name)
	}
	return obj.(*api.Certificate), nil
}

func (c *Controller) projectCertificate(r *api.Certificate, projections map[string]ioutilz.FileProjection) error {
	pemCrt, pemKey, err := c.store.Get(r)
	if err != nil {
		return err
	}
	projections["tls/"+r.SecretName()+".pem"] = ioutilz.FileProjection{Mode: 0755, Data: certificateToPEMData(pemCrt, pemKey)}
	return nil
}
