package operator

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/tools/queue"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initDeploymentWatcher() {
	op.dpInformer = op.kubeInformerFactory.Apps().V1beta1().Deployments().Informer()
	op.dpQueue = queue.New("Deployment", op.MaxNumRequeues, op.NumThreads, op.reconcileDeployment)
	op.dpInformer.AddEventHandler(queue.NewDeleteHandler(op.dpQueue.GetQueue()))
	op.dpLister = op.kubeInformerFactory.Apps().V1beta1().Deployments().Lister()
}

func (op *Operator) reconcileDeployment(key string) error {
	_, exists, err := op.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Deployment %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreDeployment(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-deployment
func (op *Operator) restoreDeployment(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
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
