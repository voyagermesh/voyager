package operator

import (
	"reflect"

	"github.com/appscode/errors"
	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	tapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/appscode/voyager/pkg/monitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) WatchIngressTPRs() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.ExtClient.Ingresses(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&tapi.Ingress{},
		op.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if engress, ok := obj.(*tapi.Ingress); ok {
					log.Infof("%s %s@%s added", engress.GroupVersionKind(), engress.Name, engress.Namespace)
					if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", engress.GroupVersionKind(), engress.Name, engress.Namespace)
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
					log.Infof("%s %s@%s has unchanged spec and annotations", newEngress.GroupVersionKind(), newEngress.Name, newEngress.Namespace)
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
	ctrl.Run(wait.NeverStop)
}

func (op *Operator) AddEngress(engress *tapi.Ingress) {
	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.Opt, engress)
	if ctrl.IsExists() {
		// Loadbalancer resource for this ingress is found in its place,
		// so no need to create the resources. First trying to update
		// the configMap only for the rules.
		// In case of any failure in soft update we will make hard update
		// to the resource. If hard update encounters errors then we will
		// recreate the resource from scratch.
		log.Infoln("Loadbalancer is exists, trying to update")

		if svc, err := op.KubeClient.CoreV1().Services(engress.Namespace).Get(engress.OffshootName(), metav1.GetOptions{}); err == nil {
			// check port
			curPorts := make(map[int]apiv1.ServicePort)
			for _, p := range svc.Spec.Ports {
				curPorts[int(p.Port)] = p
			}

			var updateFW bool
			for svcPort, targetPort := range ctrl.Ports {
				if sp, ok := curPorts[svcPort]; !ok || sp.TargetPort.IntValue() != targetPort {
					updateFW = true // new port has to be opened
					break
				} else {
					delete(curPorts, svcPort)
				}
			}
			if len(curPorts) > 0 {
				updateFW = true // additional port was open previously
			}

			if updateFW {
				ctrl.Update(ingress.UpdateFirewall | ingress.UpdateStats)
			} else {
				ctrl.Update(ingress.UpdateConfig | ingress.UpdateStats)
			}
		} else {
			log.Warningln("Loadbalancer is exists but Soft Update failed. Retrying Hard Update")
			restartErr := ctrl.Update(ingress.RestartHAProxy)
			if restartErr != nil {
				log.Warningln("Loadbalancer is exists, But Hard Update is also failed, recreating with a cleanup")
				ctrl.Create()
			}
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
		op.ensureServiceAnnotations(engress, svc)
	}

	err := certificate.NewController(op.KubeClient, op.ExtClient, op.Opt, nil).HandleIngress(engress)
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

	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.Opt, newEngress)

	if oldHandled && !newHandled {
		ctrl.Delete()
	} else {
		if ctrl.IsExists() {
			var updateMode ingress.UpdateMode

			// Ingress Annotations Changed, Apply Changes to Targets
			// The following method do not update to HAProxy config or restart pod. It only sets the annotations
			// to the required targets.
			ctrl.UpdateTargetAnnotations(oldEngress, newEngress)

			if oldEngress.IsKeepSourceChanged(*newEngress, op.Opt.CloudProvider) {
				updateMode |= ingress.UpdateConfig
			}
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
						promCtrl := monitor.NewPrometheusController(ctrl.KubeClient, ctrl.PromClient)
						err := promCtrl.UpdateMonitor(ctrl.Ingress, oldMonSpec, newMonSpec)
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

			if oldEngress.IsPortChanged(*newEngress) || oldEngress.IsLoadBalancerSourceRangeChanged(*newEngress) {
				updateMode |= ingress.UpdateFirewall
			} else if oldEngress.IsSecretChanged(*newEngress) {
				updateMode |= ingress.RestartHAProxy
			} else {
				updateMode |= ingress.UpdateConfig
			}
			if updateMode > 0 {
				// For ingress update update HAProxy once
				ctrl.Update(updateMode)
			}
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
		op.ensureServiceAnnotations(newEngress, svc)
	}

	err := certificate.NewController(op.KubeClient, op.ExtClient, op.Opt, nil).HandleIngress(newEngress)
	if err != nil {
		log.Error(err)
	}
}

func (op *Operator) DeleteEngress(engress *tapi.Ingress) {
	ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.Opt, engress)
	ctrl.Delete()

	for _, meta := range engress.BackendServices() {
		svc, err := op.KubeClient.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		op.ensureServiceAnnotations(engress, svc)
	}
}
