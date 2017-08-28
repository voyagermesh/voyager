package operator

import (
	"reflect"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	tapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/appscode/voyager/pkg/monitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initIngressTPRWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.ExtClient.Ingresses(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&tapi.Ingress{},
		op.Opt.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if engress, ok := obj.(*tapi.Ingress); ok {
					log.Infof("%s %s@%s added", engress.GroupVersionKind(), engress.Name, engress.Namespace)
					if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", engress.GroupVersionKind(), engress.Name, engress.Namespace)
						return
					}
					if err := engress.IsValid(op.Opt.CloudProvider); err != nil {
						op.recorder.Eventf(
							engress,
							apiv1.EventTypeWarning,
							eventer.EventReasonIngressInvalid,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					go analytics.Send(engress.GroupVersionKind().String(), "ADD", "success")

					op.AddEngress(engress)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldEngress, ok := old.(*tapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object").Err())
					return
				}
				newEngress, ok := new.(*tapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object").Err())
					return
				}
				if changed, _ := oldEngress.HasChanged(*newEngress); !changed {
					return
				}
				log.Infof("%s %s@%s has changed", newEngress.GroupVersionKind(), newEngress.Name, newEngress.Namespace)
				if err := newEngress.IsValid(op.Opt.CloudProvider); err != nil {
					op.recorder.Eventf(
						newEngress,
						apiv1.EventTypeWarning,
						eventer.EventReasonIngressInvalid,
						"Reason: %s",
						err.Error(),
					)
					return
				}
				op.UpdateEngress(oldEngress, newEngress)
			},
			DeleteFunc: func(obj interface{}) {
				if engress, ok := obj.(*tapi.Ingress); ok {
					log.Infof("%s %s@%s deleted", engress.GroupVersionKind(), engress.Name, engress.Namespace)
					if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", engress.GroupVersionKind(), engress.Name, engress.Namespace)
						return
					}
					go analytics.Send(engress.GroupVersionKind().String(), "DELETE", "success")
					op.DeleteEngress(engress)
				}
			},
		},
	)
	return informer
}

func (op *Operator) AddEngress(engress *tapi.Ingress) {
	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, engress)
	if ctrl.IsExists() {
		if err := ctrl.Update(ingress.UpdateStats, nil); err != nil {
			log.Errorln(err)
		}
		return
	} else {
		ctrl.Create()
	}

	for _, meta := range engress.BackendServices() {
		svc, err := op.KubeClient.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		op.ensureEgressAnnotations(engress, svc)
	}

	err := certificate.NewController(op.KubeConfig, op.KubeClient, op.ExtClient, op.Opt, nil).HandleIngress(engress)
	if err != nil {
		log.Error(err)
	}
}

func (op *Operator) UpdateEngress(oldEngress, newEngress *tapi.Ingress) {
	oldHandled := oldEngress.ShouldHandleIngress(op.Opt.IngressClass)
	newHandled := newEngress.ShouldHandleIngress(op.Opt.IngressClass)
	if !oldHandled && !newHandled {
		return
	}

	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, newEngress)
	if oldHandled && !newHandled {
		ctrl.Delete()
	} else {
		if ctrl.IsExists() {
			var updateMode ingress.UpdateMode

			if oldEngress.IsStatsChanged(*newEngress) {
				updateMode |= ingress.UpdateStats
				if oldEngress.IsStatsSecretChanged(*newEngress) {
					updateMode |= ingress.UpdateRBAC
				}
			}
			// Check for changes in ingress.appscode.com/monitoring-agent
			if newMonSpec, newErr := newEngress.MonitorSpec(); newErr == nil {
				if oldMonSpec, oldErr := oldEngress.MonitorSpec(); oldErr == nil {
					if !reflect.DeepEqual(oldMonSpec, newMonSpec) {
						promCtrl := monitor.NewPrometheusController(op.KubeClient, op.PromClient)
						err := promCtrl.UpdateMonitor(newEngress, oldMonSpec, newMonSpec)
						if err != nil {
							return
						}
					}
					if (oldMonSpec == nil && newMonSpec != nil) ||
						(oldMonSpec != nil && newMonSpec == nil) {
						updateMode |= ingress.UpdateStats
					}
				}
			}

			// For ingress update update HAProxy once
			ctrl.Update(updateMode, oldEngress)
		} else {
			ctrl.Create()
		}
	}

	backends := map[string]metav1.ObjectMeta{}
	for k, v := range oldEngress.BackendServices() {
		backends[k] = v
	}
	for k, v := range newEngress.BackendServices() {
		backends[k] = v
	}
	for _, meta := range backends {
		svc, err := op.KubeClient.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		op.ensureEgressAnnotations(newEngress, svc)
	}

	err := certificate.NewController(op.KubeConfig, op.KubeClient, op.ExtClient, op.Opt, nil).HandleIngress(newEngress)
	if err != nil {
		log.Error(err)
	}
}

func (op *Operator) DeleteEngress(engress *tapi.Ingress) {
	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, engress)
	ctrl.Delete()

	for _, meta := range engress.BackendServices() {
		svc, err := op.KubeClient.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		op.ensureEgressAnnotations(engress, svc)
	}
}
