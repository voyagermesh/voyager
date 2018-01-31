package operator

import (
	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ext_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initIngressWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.ingQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress")

	op.ingIndexer, op.ingInformer = cache.NewIndexerInformer(lw, &extensions.Ingress{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			engress, err := api.NewEngressFromIngress(obj.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Ingress. Reason %v", engress.Namespace, engress.Name, err)
				return
			}
			if err := engress.IsValid(op.Opt.CloudProvider); err != nil {
				op.recorder.Eventf(
					engress.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonIngressInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				op.ingQueue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldEngress, err := api.NewEngressFromIngress(old.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Engress. Reason %v", oldEngress.Namespace, oldEngress.Name, err)
				return
			}
			newEngress, err := api.NewEngressFromIngress(new.(*extensions.Ingress))
			if err != nil {
				log.Errorf("Failed to convert Ingress %s/%s into Engress. Reason %v", newEngress.Namespace, newEngress.Name, err)
				return
			}

			if changed, _ := oldEngress.HasChanged(*newEngress); !changed {
				return
			}
			diff := meta.Diff(oldEngress, newEngress)
			log.Infof("%s %s/%s has changed. Diff: %s", newEngress.GroupVersionKind(), newEngress.Namespace, newEngress.Name, diff)

			if err := newEngress.IsValid(op.Opt.CloudProvider); err != nil {
				op.recorder.Eventf(
					newEngress.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonIngressInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}

			if key, err := cache.MetaNamespaceKeyFunc(new); err == nil {
				op.ingQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
				op.ingQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.ingLister = ext_listers.NewIngressLister(op.ingIndexer)
}

func (op *Operator) runIngressWatcher() {
	for op.processNextIngress() {
	}
}

func (op *Operator) processNextIngress() bool {
	key, quit := op.ingQueue.Get()
	if quit {
		return false
	}
	defer op.ingQueue.Done(key)

	err := op.reconcileIngress(key.(string))
	if err == nil {
		op.ingQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process ingress %v. Reason: %s", key, err)

	if op.ingQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing ingress %v: %v", key, err)
		op.ingQueue.AddRateLimited(key)
		return true
	}

	op.ingQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping ingress %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) reconcileIngress(key string) error {
	obj, exists, err := op.ingIndexer.GetByKey(key)
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
	ctrl := ingress.NewController(etx.Background(), op.KubeClient, op.CRDClient, op.VoyagerClient, op.PromClient, op.svcLister, op.epLister, op.Opt, engress)

	if ing.DeletionTimestamp != nil {
		if core_util.HasFinalizer(ing.ObjectMeta, api.VoyagerFinalizer) {
			glog.Infof("Delete for engress %s\n", key)
			ctrl.Delete()
			ext_util.PatchIngress(op.KubeClient, ing, func(obj *extensions.Ingress) *extensions.Ingress {
				core_util.RemoveFinalizer(obj.ObjectMeta, api.VoyagerFinalizer)
				return obj
			})
		}
	} else {
		glog.Infof("Sync/Add/Update for ingress %s\n", key)
		if !core_util.HasFinalizer(ing.ObjectMeta, api.VoyagerFinalizer) {
			ext_util.PatchIngress(op.KubeClient, ing, func(obj *extensions.Ingress) *extensions.Ingress {
				core_util.AddFinalizer(obj.ObjectMeta, api.VoyagerFinalizer)
				return obj
			})
		}
		if engress.ShouldHandleIngress(op.Opt.IngressClass) {
			return ctrl.Reconcile()
		} else {
			log.Infof("%s %s/%s does not match ingress class", engress.APISchema(), engress.Namespace, engress.Name)
			ctrl.Delete()
		}
	}
	return nil
}
