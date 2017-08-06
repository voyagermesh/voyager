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

func (c *Controller) Update(mode UpdateMode) error {
	err := c.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// Update HAProxy config
	err = c.updateConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if mode&UpdateFirewall > 0 ||
		mode&RestartHAProxy > 0 ||
		mode&UpdateStats > 0 {
		err := c.recreatePods()
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressUpdateFailed,
				"Failed to update Pods, %s", err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressUpdateSuccessful,
			"Successfully updated Pods",
		)
	}

	if mode&UpdateFirewall > 0 {
		err := c.updateLBSvc()
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceUpdateFailed,
				"Failed to update LBService, %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceUpdateSuccessful,
			"Successfully updated LBService",
		)

		go c.updateStatus()
	}

	if mode&UpdateStats > 0 {
		if c.Ingress.Stats() {
			err := c.ensureStatsService()
			if err != nil {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceCreateFailed,
					"Failed to create Stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceCreateSuccessful,
					"Successfully created Stats Service %s",
					c.Ingress.StatsServiceName(),
				)
			}
		} else {
			err := c.ensureStatsServiceDeleted()
			if err != nil {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceDeleteFailed,
					"Failed to delete Stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceDeleteSuccessful,
					"Successfully deleted Stats Service %s",
					c.Ingress.StatsServiceName(),
				)
			}
		}
	}

	if mode&UpdateRBAC > 0 {
		c.ensureRoles()
	}

	return nil
}

func (c *Controller) updateConfigMap() error {
	cMap, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
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
		cMap.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		cMap.Annotations[api.OriginName] = c.Ingress.GetName()
	}

	if cMap.Data["haproxy.cfg"] != c.HAProxyConfig {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = c.HAProxyConfig

		_, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Update(cMap)
		if err != nil {
			c.recorder.Eventf(c.Ingress, apiv1.EventTypeWarning, "ConfigMapUpdateFailed", "HAProxy configuration Update failed, %s", err.Error())
			return errors.FromErr(err).Err()
		}
		// Add event only if the ConfigMap Really Updated
		c.recorder.Eventf(c.Ingress, apiv1.EventTypeNormal, "ConfigMapUpdated", "ConfigMap Updated, HAProxy will restart itself now via reloader")
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (c *Controller) recreatePods() error {
	if !c.SupportsLBType() {
		return errors.Newf("LBType %s is unsupported for cloud provider: %s", c.Ingress.LBType(), c.Opt.CloudProvider).Err()
	}

	if c.Ingress.LBType() == api.LBTypeHostPort {
		err := c.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = c.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if c.Ingress.LBType() == api.LBTypeNodePort {
		err := c.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = c.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		// Ignore Error.
		c.deleteResidualPods()
		err := c.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = c.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (c *Controller) updateLBSvc() error {
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	curPorts := make(map[int32]apiv1.ServicePort)
	for _, p := range svc.Spec.Ports {
		curPorts[p.Port] = p
	}
	svc.Spec.Ports = make([]apiv1.ServicePort, 0)
	for svcPort, target := range c.PortMapping {
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
		svc.Spec.LoadBalancerSourceRanges = c.Ingress.Spec.LoadBalancerSourceRanges
	}

	if svc.Annotations == nil {
		// This is a safety check, annotations will not be nil
		svc.Annotations = make(map[string]string)
	}
	_, sourceNameFound := svc.Annotations[api.OriginName]
	_, sourceTypeFound := svc.Annotations[api.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		// Old version object
		svc.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		svc.Annotations[api.OriginName] = c.Ingress.GetName()
	}

	svc, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	// open up firewall
	log.Infoln("Loadbalancer CloudManager", c.CloudManager, "serviceType", svc.Spec.Type)
	if (c.Ingress.LBType() == api.LBTypeHostPort) && c.CloudManager != nil {
		daemonNodes, err := c.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(c.Ingress.NodeSelector()).String(),
		})
		if err != nil {
			log.Infoln("node not found with nodeSelector, cause", err)
			return errors.FromErr(err).Err()
		}
		// open up firewall
		log.Debugln("Checking cloud manager", c.CloudManager)
		if c.CloudManager != nil {
			log.Debugln("cloud manager not nil")
			if fw, ok := c.CloudManager.Firewall(); ok {
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

func (c *Controller) UpdateTargetAnnotations(old *api.Ingress, new *api.Ingress) error {
	// Check for changes in ingress.appscode.com/annotations-service
	if newSvcAns, newOk := new.ServiceAnnotations(c.Opt.CloudProvider); newOk {
		if oldSvcAns, oldOk := old.ServiceAnnotations(c.Opt.CloudProvider); oldOk {
			if !reflect.DeepEqual(oldSvcAns, newSvcAns) {
				svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
				if err != nil {
					return errors.FromErr(err).Err()
				}
				svc.Annotations = mergeAnnotations(svc.Annotations, oldSvcAns, newSvcAns)

				svc, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(svc)
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
				if c.Ingress.LBType() == api.LBTypeHostPort {
					daemonset, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
					if err != nil {
						return errors.FromErr(err).Err()
					}
					daemonset.Spec.Template.Annotations = newPodAns
					daemonset, err = c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Update(daemonset)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					if daemonset.Spec.Selector != nil {
						pods, _ := c.KubeClient.CoreV1().Pods(c.Ingress.Namespace).List(metav1.ListOptions{
							LabelSelector: labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels).String(),
						})
						for _, pod := range pods.Items {
							pod.Annotations = mergeAnnotations(pod.Annotations, oldPodAns, newPodAns)
							_, err := c.KubeClient.CoreV1().Pods(c.Ingress.Namespace).Update(&pod)
							if err != nil {
								log.Errorln("Failed to Update Pods", err)
							}
						}
					}
				} else {
					dep, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
					if err != nil {
						return errors.FromErr(err).Err()
					}
					dep.Spec.Template.Annotations = mergeAnnotations(dep.Spec.Template.Annotations, oldPodAns, newPodAns)
					_, err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Update(dep)
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
