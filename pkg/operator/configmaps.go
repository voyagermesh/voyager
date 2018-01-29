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

func (op *Operator) initConfigMapWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.CoreV1().ConfigMaps(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().ConfigMaps(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.cfgQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "configmap")

	op.cfgIndexer, op.cfgInformer = cache.NewIndexerInformer(lw, &core.ConfigMap{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				op.cfgQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.cfgLister = core_listers.NewConfigMapLister(op.cfgIndexer)
}

func (op *Operator) runConfigMapWatcher() {
	for op.processNextConfigMap() {
	}
}

func (op *Operator) processNextConfigMap() bool {
	key, quit := op.cfgQueue.Get()
	if quit {
		return false
	}
	defer op.cfgQueue.Done(key)

	err := op.runConfigMapInjector(key.(string))
	if err == nil {
		op.cfgQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process ConfigMap %v. Reason: %s", key, err)

	if op.cfgQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing configmap %v: %v", key, err)
		op.cfgQueue.AddRateLimited(key)
		return true
	}

	op.cfgQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping configmap %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) runConfigMapInjector(key string) error {
	_, exists, err := op.cfgIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("ConfigMap %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreConfigMapIfRequired(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-configmap
func (op *Operator) restoreConfigMapIfRequired(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) && engress.Namespace == ns && engress.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(engress); err != nil {
				return err
			} else {
				op.engQueue.Add(key)
				log.Infof("Add/Delete/Update of haproxy configmap %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
