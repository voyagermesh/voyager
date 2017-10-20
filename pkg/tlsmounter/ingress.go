package tlsmounter

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	extension "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Controller) initIngressWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.ExtensionsV1beta1().Ingresses(c.options.IngressRef.Namespace).List(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.ExtensionsV1beta1().Ingresses(c.options.IngressRef.Namespace).Watch(metav1.ListOptions{
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
	c.ingIndexer, c.ingInformer = cache.NewIndexerInformer(lw, &extension.Ingress{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if _, ok := obj.(*extension.Ingress); !ok {
				return
			}
			if r, err := api.NewEngressFromIngress(obj); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						c.ingQueue.Add(key)
					}
				}
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			if _, ok := new.(*extension.Ingress); !ok {
				return
			}
			if r, err := api.NewEngressFromIngress(new); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						c.ingQueue.Add(key)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if _, ok := obj.(*extension.Ingress); !ok {
				return
			}
			if r, err := api.NewEngressFromIngress(obj); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.ingQueue.Add(key)
				}
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) runIngressWatcher() {
	for c.processNextIngress() {
	}
}

func (c *Controller) processNextIngress() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.ingQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.ingQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.syncIngress(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.ingQueue.Forget(key)
		return true
	}
	log.Errorln("Failed to process Ingress %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.ingQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.ingQueue.AddRateLimited(key)
		return true
	}

	c.ingQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncIngress(key string) error {
	obj, exists, err := c.ingIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Ingress, so that we will see a delete for one d
		fmt.Printf("Ingress %s does not exist anymore\n", key)
	} else {
		d, err := api.NewEngressFromIngress(obj)
		if err != nil {
			return err
		}
		fmt.Printf("Sync/Add/Update for Ingress %s\n", d.GetName())

		err = c.mountIngress(d, true)
		if err != nil {
			c.recorder.Event(
				d.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressTLSMountFailed,
				err.Error(),
			)
			return err
		}
	}
	return nil
}
