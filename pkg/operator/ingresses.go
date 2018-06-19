package operator

import (
	"context"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/voyager/apis/voyager"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initIngressWatcher() {
	op.ingInformer = op.kubeInformerFactory.Extensions().V1beta1().Ingresses().Informer()
	op.ingQueue = queue.New("Ingress", op.MaxNumRequeues, op.NumThreads, op.reconcileIngress)
	op.ingInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			engress, err := api.NewEngressFromIngress(obj.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Ingress. Reason %v", engress.Namespace, engress.Name, err)
				return
			}
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
			queue.Enqueue(op.ingQueue.GetQueue(), obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old, err := api.NewEngressFromIngress(oldObj.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Engress. Reason %v", old.Namespace, old.Name, err)
				return
			}
			nu, err := api.NewEngressFromIngress(newObj.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Engress. Reason %v", nu.Namespace, nu.Name, err)
				return
			}

			if changed, _ := old.HasChanged(*nu); !changed {
				return
			}
			diff := meta.Diff(old, nu)
			log.Infof("%s %s/%s has changed. Diff: %s", nu.GroupVersionKind(), nu.Namespace, nu.Name, diff)

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
			queue.Enqueue(op.ingQueue.GetQueue(), newObj)
		},
	})
	op.ingLister = op.kubeInformerFactory.Extensions().V1beta1().Ingresses().Lister()
}

func (op *Operator) reconcileIngress(key string) error {
	obj, exists, err := op.ingInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Ingress %s does not exist anymore\n", key)
		return nil
	}

	ing := obj.(*extensions.Ingress).DeepCopy()
	engress, err := api.NewEngressFromIngress(ing)
	if err != nil {
		log.Errorf("Failed to convert Ingress %s/%s into Ingress. Reason %v", engress.Namespace, engress.Name, err)
		return nil
	}

	ctrl := ingress.NewController(NewID(context.Background()), op.KubeClient, op.WorkloadClient, op.CRDClient, op.VoyagerClient, op.PromClient, op.svcLister, op.epLister, op.Config, engress, op.recorder)

	if ing.DeletionTimestamp != nil {
		if core_util.HasFinalizer(ing.ObjectMeta, voyager.GroupName) {
			glog.Infof("Delete for engress %s\n", key)
			ctrl.Delete()
			ext_util.PatchIngress(op.KubeClient, ing, func(obj *extensions.Ingress) *extensions.Ingress {
				obj.ObjectMeta = core_util.RemoveFinalizer(obj.ObjectMeta, voyager.GroupName)
				return obj
			})
		}
	} else {
		glog.Infof("Sync/Add/Update for ingress %s\n", key)
		if !core_util.HasFinalizer(ing.ObjectMeta, voyager.GroupName) {
			ext_util.PatchIngress(op.KubeClient, ing, func(obj *extensions.Ingress) *extensions.Ingress {
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
