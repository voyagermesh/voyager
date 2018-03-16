package operator

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initServiceWatcher() {
	op.svcInformer = op.kubeInformerFactory.Core().V1().Services().Informer()
	op.svcQueue = queue.New("Service", op.MaxNumRequeues, op.NumThreads, op.reconcileService)
	op.svcInformer.AddEventHandler(queue.DefaultEventHandler(op.svcQueue.GetQueue()))
	op.svcLister = op.kubeInformerFactory.Core().V1().Services().Lister()
}

func (op *Operator) reconcileService(key string) error {
	obj, exists, err := op.svcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Service %s does not exist anymore\n", key)
		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		if isOffshoot, err := op.restoreIngressService(name, ns); isOffshoot {
			return err // assume offshoot service can't be backend service
		} else {
			return op.updateHAProxyConfig(name, ns)
		}
	} else {
		svc := obj.(*core.Service).DeepCopy()
		glog.Infof("Sync/Add/Update for Service %s\n", svc.GetName())
		return op.updateHAProxyConfig(svc.Name, svc.Namespace)
	}
	return nil
}

// requeue ingress if offshoot-service deleted
// return true if service is offshoot for any ingress
func (op *Operator) restoreIngressService(name, ns string) (bool, error) {
	items, err := op.listIngresses()
	if err == nil {
		for i := range items {
			ing := &items[i]
			if ing.DeletionTimestamp == nil &&
				ing.ShouldHandleIngress(op.IngressClass) &&
				ing.Namespace == ns &&
				ing.OffshootName() == name {
				key, err := cache.MetaNamespaceKeyFunc(ing)
				if err == nil {
					op.getIngressQueue(ing.APISchema()).Add(key)
					log.Infof("Add/Delete/Update of offshoot service %s/%s, Ingress %s re-queued for update", ns, name, key)
				}
				return true, err
			}
		}
	}
	return false, err
}

// requeue ingress if add/delete/update of backend-service
func (op *Operator) updateHAProxyConfig(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.HasBackendService(name, ns) {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err == nil {
				op.getIngressQueue(ing.APISchema()).Add(key)
				log.Infof("Add/Delete/Update of backend service %s/%s, Ingress %s re-queued for update", ns, name, key)
			}
		}
	}
	return nil
}

func (op *Operator) findOrigin(meta metav1.ObjectMeta) (*api.Ingress, error) {
	if meta.Annotations == nil {
		return nil, nil
	}

	sourceName, sourceNameFound := meta.Annotations[api.OriginName]
	sourceType, sourceTypeFound := meta.Annotations[api.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		return nil, errors.New("no Types or Name found")
	}

	if sourceType == api.APISchemaIngress {
		ingress, err := op.KubeClient.ExtensionsV1beta1().Ingresses(meta.Namespace).Get(sourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return api.NewEngressFromIngress(ingress)
	} else if sourceType == api.APISchemaEngress {
		return op.VoyagerClient.VoyagerV1beta1().Ingresses(meta.Namespace).Get(sourceName, metav1.GetOptions{})
	}
	return nil, errors.Errorf("unknown ingress type %s", sourceType)
}
