package operator

import (
	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/google/go-cmp/cmp"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initIngresseWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&extensions.Ingress{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if ingress, ok := obj.(*extensions.Ingress); ok {
					ctx := etx.Background()
					logger := log.New(ctx)
					logger.Infof("%s %s@%s added", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					engress, err := api.NewEngressFromIngress(ingress)
					if err != nil {
						logger.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", ingress.Name, ingress.Namespace, err)
						return
					}
					if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
						logger.Infof("%s %s@%s does not match ingress class", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)
						return
					}
					if err := engress.IsValid(op.Opt.CloudProvider); err != nil {
						op.recorder.Eventf(
							engress.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonIngressInvalid,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					op.AddEngress(ctx, engress)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				ctx := etx.Background()
				logger := log.New(ctx)
				oldIngress, ok := old.(*extensions.Ingress)
				if !ok {
					logger.Errorln("Invalid Ingress object")
					return
				}
				newIngress, ok := new.(*extensions.Ingress)
				if !ok {
					logger.Errorln("Invalid Ingress object")
					return
				}

				oldEngress, err := api.NewEngressFromIngress(oldIngress)
				if err != nil {
					logger.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", oldIngress.Name, oldIngress.Namespace, err)
					return
				}
				newEngress, err := api.NewEngressFromIngress(newIngress)
				if err != nil {
					logger.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", newIngress.Name, newIngress.Namespace, err)
					return
				}
				if changed, _ := oldEngress.HasChanged(*newEngress); !changed {
					return
				}
				diff := cmp.Diff(oldEngress, newEngress, cmp.Comparer(func(x, y resource.Quantity) bool {
					return x.Cmp(y) == 0
				}))
				logger.Infof("%s %s@%s has changed. Diff: %s", newIngress.GroupVersionKind(), newIngress.Name, newIngress.Namespace, diff)
				if err := newEngress.IsValid(op.Opt.CloudProvider); err != nil {
					op.recorder.Eventf(
						newEngress.ObjectReference(),
						apiv1.EventTypeWarning,
						eventer.EventReasonIngressInvalid,
						"Reason: %s",
						err.Error(),
					)
					return
				}
				op.UpdateEngress(ctx, oldEngress, newEngress)
			},
			DeleteFunc: func(obj interface{}) {
				if ingress, ok := obj.(*extensions.Ingress); ok {
					ctx := etx.Background()
					logger := log.New(ctx)
					logger.Infof("%s %s@%s deleted", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					engress, err := api.NewEngressFromIngress(ingress)
					if err != nil {
						logger.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", ingress.Name, ingress.Namespace, err)
						return
					}
					if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
						logger.Infof("%s %s@%s does not match ingress class", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)
						return
					}
					op.DeleteEngress(ctx, engress)
				}
			},
		},
	)
	return informer
}
