package operator

import (
	"github.com/appscode/go/log"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/tools/queue"
	wpi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (op *Operator) initStatefulSetWatcher() {
	op.stsInformer = op.kubeInformerFactory.Apps().V1().StatefulSets().Informer()
	op.stsQueue = queue.New("StatefulSet", op.MaxNumRequeues, op.NumThreads, op.reconcileStatefulSet)
	op.stsInformer.AddEventHandler(queue.NewDeleteHandler(op.dpQueue.GetQueue()))
	op.stsLister = op.kubeInformerFactory.Apps().V1().StatefulSets().Lister()
}

func (op *Operator) reconcileStatefulSet(key string) error {
	_, exists, err := op.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("StatefulSet %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreStatefulSet(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-statefulset
func (op *Operator) restoreStatefulSet(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.WorkloadKind() == wpi.KindStatefulSet &&
			ing.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				log.Infof("Add/Delete/Update of haproxy statefulset %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
