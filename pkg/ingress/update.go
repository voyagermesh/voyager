package ingress

import (
	"reflect"
	"strconv"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type updateType int

const (
	UpdateConfig   updateType = 1 << iota // only reset haproxy config
	RestartHAProxy                        // secret changes, ports unchanged
	UpdateFirewall                        // ports changed
	UpdateStats                           // Update things for stats update
)

func (lbc *EngressController) Update(t updateType) error {
	log.Debugln("updating engress specs with type", t)
	lbc.parse()
	err := lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// Update HAProxy config
	err = lbc.updateConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if t&UpdateFirewall > 0 ||
		t&RestartHAProxy > 0 ||
		t&UpdateStats > 0 {
		err := lbc.recreatePods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	if t&UpdateFirewall > 0 {
		err := lbc.updateLBSvc()
		if err != nil {
			// Only update if the service is updated.
			go lbc.updateStatus()
		}
		return err
	}

	if t&UpdateStats > 0 {
		if lbc.Parsed.Stats {
			lbc.ensureStatsService()
		} else {
			lbc.ensureStatsServiceDeleted()
		}
	}

	return nil
}

func (lbc *EngressController) updateConfigMap() error {
	cMap, err := lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
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
		cMap.Annotations[api.OriginAPISchema] = lbc.Resource.APISchema()
		cMap.Annotations[api.OriginName] = lbc.Resource.GetName()
	}

	if cMap.Data["haproxy.cfg"] != lbc.ConfigData {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = lbc.ConfigData

		_, err := lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Update(cMap)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (lbc *EngressController) recreatePods() error {
	if lbc.Resource.LBType() == api.LBTypeDaemon || lbc.Resource.LBType() == api.LBTypeHostPort {
		err := lbc.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		err = lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Resource.LBType() == api.LBTypeNodePort {
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		err = lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		if lbc.SupportsLoadBalancerType() {
			// Ignore Error.
			lbc.deleteResidualPods()
			err := lbc.deleteNodePortPods()
			if err != nil {
				return errors.FromErr(err).Err()
			}
			err = lbc.createNodePortPods()
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			return errors.New("LoadBalancer type ingress is unsupported for cloud provider:", lbc.ProviderName).Err()
		}
	}
	return nil
}

func (lbc *EngressController) updateLBSvc() error {
	svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	curPorts := make(map[int32]apiv1.ServicePort)
	for _, p := range svc.Spec.Ports {
		curPorts[p.Port] = p
	}
	svc.Spec.Ports = make([]apiv1.ServicePort, 0)
	for targetPort, svcPort := range lbc.Ports {
		if sp, found := curPorts[int32(svcPort)]; found && sp.TargetPort.IntValue() == targetPort {
			svc.Spec.Ports = append(svc.Spec.Ports, sp)
		} else {
			svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
				Name:       "tcp-" + strconv.Itoa(svcPort),
				Protocol:   "TCP",
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt(targetPort),
			})
		}
	}

	if svc.Spec.Type == apiv1.ServiceTypeLoadBalancer {
		// Update Source Range
		svc.Spec.LoadBalancerSourceRanges = lbc.Resource.Spec.LoadBalancerSourceRanges
	}

	if svc.Annotations == nil {
		// This is a safety check, annotations will not be nil
		svc.Annotations = make(map[string]string)
	}
	_, sourceNameFound := svc.Annotations[api.OriginName]
	_, sourceTypeFound := svc.Annotations[api.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		// Old version object
		svc.Annotations[api.OriginAPISchema] = lbc.Resource.APISchema()
		svc.Annotations[api.OriginName] = lbc.Resource.GetName()
	}

	svc, err = lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Update(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	// open up firewall
	log.Infoln("Loadbalancer CloudManager", lbc.CloudManager, "serviceType", svc.Spec.Type)
	if (lbc.Resource.LBType() == api.LBTypeDaemon || lbc.Resource.LBType() == api.LBTypeHostPort) && lbc.CloudManager != nil {
		daemonNodes, err := lbc.KubeClient.Core().Nodes().List(apiv1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set(lbc.Resource.NodeSelector())),
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
				convertedSvc := &apiv1.Service{}
				apiv1.Scheme.Convert(svc, convertedSvc, nil)
				for _, node := range daemonNodes.Items {
					err = fw.EnsureFirewall(convertedSvc, node.Name)
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

func (lbc *EngressController) UpdateTargetAnnotations(old *api.Ingress, new *api.Ingress) error {
	lbc.parse()

	// Check for changes in ingress.appscode.com/annotations-service
	if newSvcAns, newOk := new.ServiceAnnotations(lbc.ProviderName); newOk {
		if oldSvcAns, oldOk := old.ServiceAnnotations(lbc.ProviderName); oldOk {
			if !reflect.DeepEqual(oldSvcAns, newSvcAns) {
				svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
				if err != nil {
					return errors.FromErr(err).Err()
				}
				svc.Annotations = mergeAnnotations(svc.Annotations, oldSvcAns, newSvcAns)

				svc, err = lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Update(svc)
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
				if lbc.Resource.LBType() == api.LBTypeDaemon || lbc.Resource.LBType() == api.LBTypeHostPort {
					daemonset, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
					if err != nil {
						return errors.FromErr(err).Err()
					}
					daemonset.Spec.Template.Annotations = newPodAns
					daemonset, err = lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Update(daemonset)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					if daemonset.Spec.Selector != nil {
						pods, _ := lbc.KubeClient.Core().Pods(lbc.Resource.Namespace).List(apiv1.ListOptions{
							LabelSelector: labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels),
						})
						for _, pod := range pods.Items {
							pod.Annotations = mergeAnnotations(pod.Annotations, oldPodAns, newPodAns)
							_, err := lbc.KubeClient.Core().Pods(lbc.Resource.Namespace).Update(&pod)
							if err != nil {
								log.Errorln("Failed to Update Pods", err)
							}
						}
					}
				} else {
					dep, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
					if err != nil {
						return errors.FromErr(err).Err()
					}
					dep.Spec.Template.Annotations = mergeAnnotations(dep.Spec.Template.Annotations, oldPodAns, newPodAns)
					_, err = lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Update(dep)
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
