package operator

import (
	"github.com/appscode/go/log"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initNamespaceWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.CoreV1().Namespaces().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Namespaces().Watch(options)
		},
	}

	// create the workqueue
	op.nsQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "namespace")

	op.nsIndexer, op.nsInformer = cache.NewIndexerInformer(lw, &core.Namespace{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
				op.nsQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.nsLister = core_listers.NewNamespaceLister(op.nsIndexer)
}

func (op *Operator) runNamespaceWatcher() {
	for op.processNextNamespace() {
	}
}

func (op *Operator) processNextNamespace() bool {
	key, quit := op.nsQueue.Get()
	if quit {
		return false
	}
	defer op.nsQueue.Done(key)

	err := op.runNamespaceInjector(key.(string))
	if err == nil {
		op.nsQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Namespace %v. Reason: %s", key, err)

	if op.nsQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing namespace %v: %v", key, err)
		op.nsQueue.AddRateLimited(key)
		return true
	}

	op.nsQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping namespace %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) runNamespaceInjector(key string) error {
	_, exists, err := op.nsIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Namespace %s does not exist anymore\n", key)
		if _, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			op.deleteCRDs(name)
		}
	}
	return nil
}

func (op *Operator) deleteCRDs(ns string) {
	if resources, err := op.VoyagerClient.Certificates(ns).List(metav1.ListOptions{}); err == nil {
		for _, resource := range resources.Items {
			op.VoyagerClient.Certificates(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	}
	if resources, err := op.VoyagerClient.Ingresses(ns).List(metav1.ListOptions{}); err == nil {
		for _, resource := range resources.Items {
			op.VoyagerClient.Ingresses(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	}
}
