package ingress

import (
	"strconv"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/intstr"
)

type updateType int

const (
	UpdateTypeSoft updateType = iota
	UpdateTypeHard
)

func (lbc *EngressController) Update(Type updateType) error {
	log.Debugln("updating engress specs with type", Type)
	lbc.parse()
	err := lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// update config config map updates an existing map data.
	err = lbc.updateConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if Type == UpdateTypeHard {
		err := lbc.hardUpdate()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}

	serviceName := VoyagerPrefix + lbc.Config.Name
	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(serviceName)
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
	} else if lbc.Options.LBType == LBLoadBalancer && lbc.CloudManager != nil {
		if svc.Spec.Type == kapi.ServiceTypeNodePort {
			log.Infof("Service Type is %s, needs to update underlying cloud loadbalancers", svc.Spec.Type)
			hosts := make([]string, 0)
			if ins, ok := lbc.CloudManager.Instances(); ok {
				nodes, _ := ins.List("")
				for _, node := range nodes {
					hosts = append(hosts, string(node))
				}
			}
			log.Infoln("Got hosts", hosts)
			if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
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
	}
	return nil
}

func (lbc *EngressController) updateConfigMap() error {
	log.Infoln()
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

func (lbc *EngressController) hardUpdate() error {
	var err error
	if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
		err = lbc.deleteHostPortPods()
	} else {
		err = lbc.deleteLoadBalancerPods()
	}
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
		err = lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		if lbc.Options.ProviderName == "aws" ||
			lbc.Options.ProviderName == "gce" ||
			lbc.Options.ProviderName == "gke" ||
			lbc.Options.ProviderName == "azure" ||
			lbc.Options.ProviderName == "minikube" {
			err = lbc.createLoadBalancerPods()
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			err = errors.New("LoadBalancer type ingress is unsupported for cloud provider:", lbc.Options.ProviderName).Err()
		}
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	svc.Spec.Ports = make([]kapi.ServicePort, 0)
	for _, port := range lbc.Options.Ports {
		p := kapi.ServicePort{
			Name:       "tcp-" + strconv.Itoa(port),
			Protocol:   "TCP",
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	svc, err = lbc.KubeClient.Core().Services(lbc.Config.Namespace).Update(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}
