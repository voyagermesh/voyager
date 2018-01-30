package operator

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initServiceMonitorWatcher() {
	if !meta.IsPreferredAPIResource(op.KubeClient, prom.Group+"/"+prom.Version, prom.ServiceMonitorsKind) {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", prom.Group+"/"+prom.Version, prom.ServiceMonitorsKind)
		return
	}

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.PromClient.ServiceMonitors(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.PromClient.ServiceMonitors(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}

	// create the workqueue
	op.monQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "service-monitor")

	op.monIndexer, op.monInformer = cache.NewIndexerInformer(lw, &prom.ServiceMonitor{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
				op.monQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (op *Operator) runServiceMonitorWatcher() {
	for op.processNextServiceMonitor() {
	}
}

func (op *Operator) processNextServiceMonitor() bool {
	key, quit := op.monQueue.Get()
	if quit {
		return false
	}
	defer op.monQueue.Done(key)

	err := op.reconcileServiceMonitor(key.(string))
	if err == nil {
		op.monQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process ServiceMonitor %v. Reason: %s", key, err)

	if op.monQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing service-monitor %v: %v", key, err)
		op.monQueue.AddRateLimited(key)
		return true
	}

	op.monQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping service-monitor %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) reconcileServiceMonitor(key string) error {
	_, exists, err := op.monIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("ServiceMonitor %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreServiceMonitor(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes service-monitor
func (op *Operator) restoreServiceMonitor(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.Opt.IngressClass) &&
			ing.Namespace == ns &&
			ing.StatsServiceName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.engQueue.Add(key)
				log.Infof("Add/Delete/Update of service-monitor %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
