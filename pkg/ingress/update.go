package ingress

import (
	"reflect"
	"strconv"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/eventer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type UpdateMode int

const (
	UpdateConfig   UpdateMode = 1 << iota // only reset haproxy config
	RestartHAProxy                        // secret changes, ports unchanged
	UpdateFirewall                        // ports changed
	UpdateStats                           // Update things for stats update
	UpdateRBAC                            // Update RBAC Roles as stats secret name is changes
)

func (lbc *Controller) Update(mode UpdateMode) error {
	err := lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// Update HAProxy config
	err = lbc.updateConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if mode&UpdateFirewall > 0 ||
		mode&RestartHAProxy > 0 ||
		mode&UpdateStats > 0 {
		err := lbc.recreatePods()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressUpdateFailed,
				"Failed to update Pods, %s", err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressUpdateSuccessful,
			"Successfully updated Pods",
		)
	}

	if mode&UpdateFirewall > 0 {
		err := lbc.updateLBSvc()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceUpdateFailed,
				"Failed to update LBService, %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceUpdateSuccessful,
			"Successfully updated LBService",
		)

		go lbc.updateStatus()
	}

	if mode&UpdateStats > 0 {
		if lbc.Ingress.Stats() {
			err := lbc.ensureStatsService()
			if err != nil {
				lbc.recorder.Eventf(
					lbc.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceCreateFailed,
					"Failed to create Stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				lbc.recorder.Eventf(
					lbc.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceCreateSuccessful,
					"Successfully created Stats Service %s",
					lbc.Ingress.StatsServiceName(),
				)
			}
		} else {
			err := lbc.ensureStatsServiceDeleted()
			if err != nil {
				lbc.recorder.Eventf(
					lbc.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceDeleteFailed,
					"Failed to delete Stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				lbc.recorder.Eventf(
					lbc.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceDeleteSuccessful,
					"Successfully deleted Stats Service %s",
					lbc.Ingress.StatsServiceName(),
				)
			}
		}
	}

	if mode&UpdateRBAC > 0 {
		lbc.ensureRoles()
	}

	return nil
}

func (lbc *Controller) updateConfigMap() error {
	cMap, err := lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if cMap.Annotations == nil {
		// This is a safety check, annotations will not be nil
		cMap.Annotations = make(map[string]string)
	}
	_, sourceNameFound := cMap.Annotations[api.OriginName]
	_, sourceTypeFound := cMap.Annotations[api.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		// Old version object
		cMap.Annotations[api.OriginAPISchema] = lbc.Ingress.APISchema()
		cMap.Annotations[api.OriginName] = lbc.Ingress.GetName()
	}

	if cMap.Data["haproxy.cfg"] != lbc.ConfigData {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = lbc.ConfigData

		_, err := lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Update(cMap)
		if err != nil {
			lbc.recorder.Eventf(lbc.Ingress, apiv1.EventTypeWarning, "ConfigMapUpdateFailed", "HAProxy configuration Update failed, %s", err.Error())
			return errors.FromErr(err).Err()
		}
		// Add event only if the ConfigMap Really Updated
		lbc.recorder.Eventf(lbc.Ingress, apiv1.EventTypeNormal, "ConfigMapUpdated", "ConfigMap Updated, HAProxy will restart itself now via reloader")
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (lbc *Controller) recreatePods() error {
	if !lbc.SupportsLBType() {
		return errors.Newf("LBType %s is unsupported for cloud provider: %s", lbc.Ingress.LBType(), lbc.Opt.CloudProvider).Err()
	}

	if lbc.Ingress.LBType() == api.LBTypeHostPort {
		err := lbc.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Ingress.LBType() == api.LBTypeNodePort {
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		// Ignore Error.
		lbc.deleteResidualPods()
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) updateLBSvc() error {
	svc, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	curPorts := make(map[int32]apiv1.ServicePort)
	for _, p := range svc.Spec.Ports {
		curPorts[p.Port] = p
	}
	svc.Spec.Ports = make([]apiv1.ServicePort, 0)
	for svcPort, target := range lbc.PortMapping {
		if sp, found := curPorts[int32(svcPort)]; found && sp.TargetPort.IntValue() == target.PodPort {
			svc.Spec.Ports = append(svc.Spec.Ports, sp)
		} else {
			svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
				Name:       "tcp-" + strconv.Itoa(svcPort),
				Protocol:   "TCP",
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt(target.PodPort),
				NodePort:   int32(target.NodePort),
			})
		}
	}

	if svc.Spec.Type == apiv1.ServiceTypeLoadBalancer {
		// Update Source Range
		svc.Spec.LoadBalancerSourceRanges = lbc.Ingress.Spec.LoadBalancerSourceRanges
	}

	if svc.Annotations == nil {
		// This is a safety check, annotations will not be nil
		svc.Annotations = make(map[string]string)
	}
	_, sourceNameFound := svc.Annotations[api.OriginName]
	_, sourceTypeFound := svc.Annotations[api.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		// Old version object
		svc.Annotations[api.OriginAPISchema] = lbc.Ingress.APISchema()
		svc.Annotations[api.OriginName] = lbc.Ingress.GetName()
	}

	svc, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Update(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	// open up firewall
	log.Infoln("Loadbalancer CloudManager", lbc.CloudManager, "serviceType", svc.Spec.Type)
	if (lbc.Ingress.LBType() == api.LBTypeHostPort) && lbc.CloudManager != nil {
		daemonNodes, err := lbc.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(lbc.Ingress.NodeSelector()).String(),
		})
		if err != nil {
			log.Infoln("node not found with nodeSelector, cause", err)
			return errors.FromErr(err).Err()
		}
		// open up firewall
		log.Debugln("Checking cloud manager", lbc.CloudManager)
		if lbc.CloudManager != nil {
			log.Debugln("cloud manager not nil")
			if fw, ok := lbc.CloudManager.Firewall(); ok {
				log.Debugln("firewalls found")
				for _, node := range daemonNodes.Items {
					err = fw.EnsureFirewall(svc, node.Name)
					if err != nil {
						log.Errorln("Failed to ensure loadbalancer for node", node.Name, "cause", err)
					}
				}
				log.Debugln("getting firewalls for cloud manager failed")
			}
		}
	}
	return nil
}

func (lbc *Controller) UpdateTargetAnnotations(old *api.Ingress, new *api.Ingress) error {
	// Check for changes in ingress.appscode.com/annotations-service
	if newSvcAns, newOk := new.ServiceAnnotations(lbc.Opt.CloudProvider); newOk {
		if oldSvcAns, oldOk := old.ServiceAnnotations(lbc.Opt.CloudProvider); oldOk {
			if !reflect.DeepEqual(oldSvcAns, newSvcAns) {
				svc, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
				if err != nil {
					return errors.FromErr(err).Err()
				}
				svc.Annotations = mergeAnnotations(svc.Annotations, oldSvcAns, newSvcAns)

				svc, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Update(svc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}

	// Check for changes in ingress.appscode.com/annotations-pod
	if newPodAns, newOk := new.PodsAnnotations(); newOk {
		if oldPodAns, oldOk := old.PodsAnnotations(); oldOk {
			if !reflect.DeepEqual(oldPodAns, newPodAns) {
				if lbc.Ingress.LBType() == api.LBTypeHostPort {
					daemonset, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
					if err != nil {
						return errors.FromErr(err).Err()
					}
					daemonset.Spec.Template.Annotations = newPodAns
					daemonset, err = lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Update(daemonset)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					if daemonset.Spec.Selector != nil {
						pods, _ := lbc.KubeClient.CoreV1().Pods(lbc.Ingress.Namespace).List(metav1.ListOptions{
							LabelSelector: labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels).String(),
						})
						for _, pod := range pods.Items {
							pod.Annotations = mergeAnnotations(pod.Annotations, oldPodAns, newPodAns)
							_, err := lbc.KubeClient.CoreV1().Pods(lbc.Ingress.Namespace).Update(&pod)
							if err != nil {
								log.Errorln("Failed to Update Pods", err)
							}
						}
					}
				} else {
					dep, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
					if err != nil {
						return errors.FromErr(err).Err()
					}
					dep.Spec.Template.Annotations = mergeAnnotations(dep.Spec.Template.Annotations, oldPodAns, newPodAns)
					_, err = lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Update(dep)
					if err != nil {
						return errors.FromErr(err).Err()
					}
				}
			}
		}
	}
	return nil
}

func mergeAnnotations(obj, old, new map[string]string) map[string]string {
	if obj == nil {
		obj = make(map[string]string)
	}

	for k := range old {
		delete(obj, k)
	}

	for k, v := range new {
		obj[k] = v
	}
	return obj
}
