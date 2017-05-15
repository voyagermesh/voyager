package ingress

import (
	"strconv"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
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
		return errors.New().WithCause(err).Err()
	}
	// update config config map updates an existing map data.
	err = lbc.updateConfigMap()
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if Type == UpdateTypeHard {
		err := lbc.hardUpdate()
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
	}

	serviceName := VoyagerPrefix + lbc.Config.Name
	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(serviceName)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	log.Infoln("Loadbalancer CloudeManager", lbc.CloudManager, "serviceType", svc.Spec.Type)
	if (svc.Spec.Type == kapi.ServiceTypeNodePort) && lbc.CloudManager != nil {
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
			log.Infoln("Loadbalancer interface found, caling UpdateLoadBalancer() with", svc, "and host", hosts)
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			err := lb.UpdateLoadBalancer(lbc.Options.ClusterName, convertedSvc, hosts)
			if err != nil {
				return errors.New().WithCause(err).Err()
			}
		}
		log.Errorln("loadbalancer interface not found, reached dead blocks.")
	}
	return nil
}

func (lbc *EngressController) updateConfigMap() error {
	log.Infoln()
	cMap, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	if cMap.Data["haproxy.cfg"] != lbc.Options.ConfigData {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = lbc.Options.ConfigData

		_, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Update(cMap)
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (lbc *EngressController) hardUpdate() error {
	var err error
	if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
		err = lbc.deleteDaemonSets()
	} else {
		err = lbc.deleteRc()
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if lbc.Options.LBType == LBDaemon || lbc.Options.LBType == LBHostPort {
		err = lbc.createHostPortLB()
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
	} else {
		if lbc.Options.ProviderName == "aws" ||
			lbc.Options.ProviderName == "gce" ||
			lbc.Options.ProviderName == "gke" ||
			lbc.Options.ProviderName == "azure" ||
			lbc.Options.ProviderName == "minikube" {
			err = lbc.createLoadBalancerLB()
			if err != nil {
				return errors.New().WithCause(err).Err()
			}
		} else {
			err = errors.New("LoadBalancer type ingress is unsupported for cloud provider:", lbc.Options.ProviderName).Err()
		}
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
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
		return errors.New().WithCause(err).Err()
	}
	return nil
}
