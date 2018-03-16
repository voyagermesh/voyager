package operator

import (
	"github.com/appscode/go/log"
	prom_util "github.com/appscode/kube-mon/prometheus/v1"
	"github.com/appscode/kutil/discovery"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initServiceMonitorWatcher() {
	if !discovery.IsPreferredAPIResource(op.KubeClient.Discovery(), prom_util.SchemeGroupVersion.String(), prom.ServiceMonitorsKind) {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", prom_util.SchemeGroupVersion.String(), prom.ServiceMonitorsKind)
		return
	}

	op.smonInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc:  op.PromClient.ServiceMonitors(op.WatchNamespace).List,
			WatchFunc: op.PromClient.ServiceMonitors(op.WatchNamespace).Watch,
		},
		&prom.ServiceMonitor{}, op.ResyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	op.smonQueue = queue.New("ServiceMonitor", op.MaxNumRequeues, op.NumThreads, op.reconcileServiceMonitor)
	op.smonInformer.AddEventHandler(queue.NewDeleteHandler(op.smonQueue.GetQueue()))
}

func (op *Operator) reconcileServiceMonitor(key string) error {
	_, exists, err := op.smonInformer.GetIndexer().GetByKey(key)
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
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.StatsServiceName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				log.Infof("Add/Delete/Update of service-monitor %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}

func (op *Operator) getIngressQueue(apiVersion string) workqueue.RateLimitingInterface {
	if apiVersion == api.APISchemaIngress {
		return op.ingQueue.GetQueue()
	}
	return op.engQueue.GetQueue()
}
