package operator

import (
	"context"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/voyager/apis/voyager"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/client/clientset/versioned/typed/voyager/v1beta1/util"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initIngressCRDWatcher() {
	op.engInformer = op.voyagerInformerFactory.Voyager().V1beta1().Ingresses().Informer()
	op.engQueue = queue.New("IngressCRD", op.MaxNumRequeues, op.NumThreads, op.reconcileEngress)
	op.engInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			engress := obj.(*api.Ingress).DeepCopy()
			engress.Migrate()

			if err := engress.IsValid(op.CloudProvider); err != nil {
				op.recorder.Eventf(
					engress.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonIngressInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			queue.Enqueue(op.engQueue.GetQueue(), obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old := oldObj.(*api.Ingress).DeepCopy()
			old.Migrate()

			nu := newObj.(*api.Ingress).DeepCopy()
			nu.Migrate()

			if changed, _ := old.HasChanged(*nu); !changed {
				return
			}
			diff := meta.Diff(old, nu)
			log.Infof("%s %s/%s has changed. Diff: %s", nu.APISchema(), nu.Namespace, nu.Name, diff)

			if err := nu.IsValid(op.CloudProvider); err != nil {
				op.recorder.Eventf(
					nu.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonIngressInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			queue.Enqueue(op.engQueue.GetQueue(), newObj)
		},
	})
	op.engLister = op.voyagerInformerFactory.Voyager().V1beta1().Ingresses().Lister()
}

func (op *Operator) reconcileEngress(key string) error {
	obj, exists, err := op.engInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Engress %s does not exist anymore\n", key)
		return nil
	}

	engress := obj.(*api.Ingress).DeepCopy()
	engress.Migrate()
	ctrl := ingress.NewController(NewID(context.Background()), op.KubeClient, op.WorkloadClient, op.CRDClient, op.VoyagerClient, op.PromClient, op.svcLister, op.epLister, op.Config, engress, op.recorder)

	if engress.DeletionTimestamp != nil {
		if core_util.HasFinalizer(engress.ObjectMeta, voyager.GroupName) {
			glog.Infof("Delete for engress %s\n", key)
			ctrl.Delete()
			util.PatchIngress(op.VoyagerClient.VoyagerV1beta1(), engress, func(obj *api.Ingress) *api.Ingress {
				obj.ObjectMeta = core_util.RemoveFinalizer(obj.ObjectMeta, voyager.GroupName)
				return obj
			})
		}
	} else {
		glog.Infof("Sync/Add/Update for engress %s\n", key)
		if !core_util.HasFinalizer(engress.ObjectMeta, voyager.GroupName) {
			util.PatchIngress(op.VoyagerClient.VoyagerV1beta1(), engress, func(obj *api.Ingress) *api.Ingress {
				obj.ObjectMeta = core_util.AddFinalizer(obj.ObjectMeta, voyager.GroupName)
				return obj
			})
		}
		if engress.ShouldHandleIngress(op.IngressClass) {
			return ctrl.Reconcile()
		} else {
			log.Infof("%s %s/%s does not match ingress class", engress.APISchema(), engress.Namespace, engress.Name)
			ctrl.Delete()
		}
	}
	return nil
}
