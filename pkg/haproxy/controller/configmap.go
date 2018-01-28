package controller

import (
	"fmt"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Controller) initConfigMapWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.CoreV1().ConfigMaps(c.options.IngressRef.Namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.CoreV1().ConfigMaps(c.options.IngressRef.Namespace).Watch(options)
		},
	}

	// create the workqueue
	c.cmQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "configmap")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the ConfigMap than the version which was responsible for triggering the update.
	c.cmIndexer, c.cmInformer = cache.NewIndexerInformer(lw, &core.ConfigMap{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*core.ConfigMap); ok {
				if c.isConfigMapUsedInIngress(r) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						c.cmQueue.Add(key)
					}
				}
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			if r, ok := new.(*core.ConfigMap); ok {
				if c.isConfigMapUsedInIngress(r) {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						c.cmQueue.Add(key)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.cmQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (c *Controller) isConfigMapUsedInIngress(s *core.ConfigMap) bool {
	return s.Name == api.VoyagerPrefix+c.options.IngressRef.Name // Ingress.OffshootName()
}

func (c *Controller) runConfigMapWatcher() {
	for c.processNextConfigMap() {
	}
}

func (c *Controller) processNextConfigMap() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.cmQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two configmaps with the same key are never processed in
	// parallel.
	defer c.cmQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.syncConfigMap(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.cmQueue.Forget(key)
		return true
	}
	log.Errorln("Failed to process ConfigMap %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.cmQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing configmap %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.cmQueue.AddRateLimited(key)
		return true
	}

	c.cmQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping configmap %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the configmap to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncConfigMap(key string) error {
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

func (c *Controller) getConfigMap(name string) (*core.ConfigMap, error) {
	obj, exists, err := c.cmIndexer.GetByKey(c.options.IngressRef.Namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(core.Resource("secret"), name)
	}
	return obj.(*core.ConfigMap), nil
}

func (c *Controller) projectHAProxyConfig(r *core.ConfigMap, projections map[string]ioutilz.FileProjection) error {
	cfg, found := r.Data["haproxy.cfg"]
	if !found {
		return fmt.Errorf("configmap %s/%s is missing haproxy.cfg", c.options.IngressRef.Namespace, r.Name)
	}
	projections["haproxy.cfg"] = ioutilz.FileProjection{Mode: 0755, Data: []byte(cfg)}
	return nil
}
