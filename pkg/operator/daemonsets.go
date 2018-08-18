package operator

import (
	"github.com/appscode/go/log"
	wpi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	"github.com/appscode/kutil/tools/queue"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initDaemonSetWatcher() {
	op.dmInformer = op.kubeInformerFactory.Apps().V1().DaemonSets().Informer()
	op.dmQueue = queue.New("DaemonSet", op.MaxNumRequeues, op.NumThreads, op.reconcileDaemonSet)
	op.dmInformer.AddEventHandler(queue.NewDeleteHandler(op.dmQueue.GetQueue()))
	op.dmLister = op.kubeInformerFactory.Apps().V1().DaemonSets().Lister()
}

func (op *Operator) reconcileDaemonSet(key string) error {
	_, exists, err := op.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("DaemonSet %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreDaemonSet(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-deployment
func (op *Operator) restoreDaemonSet(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.WorkloadKind() == wpi.KindDaemonSet &&
			ing.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				log.Infof("Add/Delete/Update of haproxy deployment %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
