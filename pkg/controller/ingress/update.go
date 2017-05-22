package ingress

import (
	"reflect"
	"strconv"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/intstr"
)

type updateType int

const (
	UpdateConfig   updateType = iota // only reset haproxy config
	RestartHAProxy                   // secret changes, ports unchanged
	UpdateFirewall                   // ports changed
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

	if t == UpdateFirewall || t == RestartHAProxy {
		err := lbc.recreatePods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	if t == UpdateFirewall {
		return lbc.updateLBSvc()
	}
	return nil
}

func (lbc *EngressController) updateConfigMap() error {
	cMap, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if cMap.Data["haproxy.cfg"] != lbc.Options.ConfigData {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = lbc.Options.ConfigData

		_, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Update(cMap)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (lbc *EngressController) recreatePods() error {
	if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
		err := lbc.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		err = lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Options.LBType == LBNodePort {
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		err = lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		if lbc.Options.SupportsLoadBalancerType() {
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
			return errors.New("LoadBalancer type ingress is unsupported for cloud provider:", lbc.Options.ProviderName).Err()
		}
	}
	return nil
}

func (lbc *EngressController) updateLBSvc() error {
	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	curPorts := make(map[int32]kapi.ServicePort)
	for _, p := range svc.Spec.Ports {
		curPorts[p.Port] = p
	}
	svc.Spec.Ports = make([]kapi.ServicePort, 0)
	for _, port := range lbc.Options.Ports {
		if sp, found := curPorts[int32(port)]; found {
			svc.Spec.Ports = append(svc.Spec.Ports, sp)
		} else {
			svc.Spec.Ports = append(svc.Spec.Ports, kapi.ServicePort{
				Name:       "tcp-" + strconv.Itoa(port),
				Protocol:   "TCP",
				Port:       int32(port),
				TargetPort: intstr.FromInt(port),
			})
		}
	}
	svc, err = lbc.KubeClient.Core().Services(lbc.Config.Namespace).Update(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	// open up firewall
	log.Infoln("Loadbalancer CloudManager", lbc.CloudManager, "serviceType", svc.Spec.Type)
	if (lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort) && lbc.CloudManager != nil {
		daemonNodes, err := lbc.KubeClient.Core().Nodes().List(kapi.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set(lbc.Options.DaemonNodeSelector)),
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
				convertedSvc := &kapi.Service{}
				kapi.Scheme.Convert(svc, convertedSvc, nil)
				for _, node := range daemonNodes.Items {
					err = fw.EnsureFirewall(convertedSvc, node.Name)
					if err != nil {
						log.Errorln("Failed to ensure loadbalancer for node", node.Name, "cause", err)
					}
				}
				log.Debugln("getting firewalls for cloud manager failed")
			}
		}
	} else if lbc.Options.LBType == LBLoadBalancer &&
		svc.Spec.Type == kapi.ServiceTypeNodePort &&
		lbc.CloudManager != nil {
		log.Infof("Service Type is %s, needs to update underlying cloud loadbalancers", svc.Spec.Type)
		if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
			// Wait for nodePort to be assigned
			timeoutAt := time.Now().Add(time.Second * 600)
			for {
				svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
				if err != nil {
					return errors.FromErr(err).Err()
				}

				nodePortReady := true
				for _, p := range svc.Spec.Ports {
					if p.NodePort <= 0 {
						nodePortReady = false
						break
					}
				}
				if nodePortReady {
					break
				}

				if time.Now().After(timeoutAt) {
					return errors.New("timed out creating node port service").Err()
				}

				log.Info("Waiting for nodeport service to be ready")

				time.Sleep(10 * time.Second)
			}

			hosts := make([]string, 0)
			if ins, ok := lbc.CloudManager.Instances(); ok {
				nodes, _ := ins.List("")
				for _, node := range nodes {
					hosts = append(hosts, string(node))
				}
			}
			log.Infoln("Got hosts", hosts)

			log.Infoln("Loadbalancer interface found, calling UpdateLoadBalancer() with", svc, "and host", hosts)
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			err := lb.UpdateLoadBalancer(lbc.Options.ClusterName, convertedSvc, hosts)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
		log.Errorln("loadbalancer interface not found, reached dead blocks.")
	}
	return nil
}

func (lbc *EngressController) UpdateTargetAnnotations(old annotation, new annotation) error {
	lbc.parse()
	if newSvcAns, newOk := new.ServiceAnnotations(); newOk {
		if oldSvcAns, oldOk := old.ServiceAnnotations(); oldOk {
			if !reflect.DeepEqual(oldSvcAns, newSvcAns) {
				svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
				if err != nil {
					return errors.FromErr(err).Err()
				}
				svc.Annotations = mergeAnnotations(svc.Annotations, oldSvcAns, newSvcAns)

				svc, err = lbc.KubeClient.Core().Services(lbc.Config.Namespace).Update(svc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}

	if newPodAns, newOk := new.PodsAnnotations(); newOk {
		if oldPodAns, oldOk := old.PodsAnnotations(); oldOk {
			if !reflect.DeepEqual(oldPodAns, newPodAns) {
				if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
					daemonset, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					daemonset.Spec.Template.Annotations = newPodAns
					daemonset, err = lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Update(daemonset)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					if daemonset.Spec.Selector != nil {
						pods, _ := lbc.KubeClient.Core().Pods(lbc.Config.Namespace).List(kapi.ListOptions{
							LabelSelector: labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels),
						})
						for _, pod := range pods.Items {
							pod.Annotations = mergeAnnotations(pod.Annotations, oldPodAns, newPodAns)
							_, err := lbc.KubeClient.Core().Pods(lbc.Config.Namespace).Update(&pod)
							if err != nil {
								log.Errorln("Failed to Update Pods", err)
							}
						}
					}
				} else {
					dep, err := lbc.KubeClient.Extensions().Deployments(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
					if err != nil {
						return errors.FromErr(err).Err()
					}
					dep.Spec.Template.Annotations = mergeAnnotations(dep.Spec.Template.Annotations, oldPodAns, newPodAns)
					_, err = lbc.KubeClient.Extensions().Deployments(lbc.Config.Namespace).Update(dep)
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
