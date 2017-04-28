package ingress

import (
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
)

type updateType int

const (
	UpdateTypeSoft updateType = iota
	UpdateTypeHard
)

func (lbc *EngressController) Update(Type updateType) error {
	log.Debugln("updating engress specs with type", Type)
	if Type == UpdateTypeHard {
		err := lbc.Delete()
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		time.Sleep(time.Second * 10)
		err = lbc.Create()
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
	} else {
		lbc.parse()
		err := lbc.generateTemplate()
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		// update config config map updates an existing map data.
		err = lbc.updateConfigMap()
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		serviceName := VoyagerPrefix + lbc.Config.Name
		svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(serviceName)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		log.Infoln("Loadbalancer CloudeManager", lbc.CloudManager, "serviceType", svc.Spec.Type)
		if svc.Spec.Type == kapi.ServiceTypeNodePort && lbc.CloudManager != nil {
			log.Infoln("Service Type is Nodeport, needs to update underlying cloude loadbalancers")
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
					return errors.New().WithCause(err).Internal()
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
		return errors.New().WithCause(err).Internal()
	}
	if cMap.Data["haproxy.cfg"] != lbc.Options.ConfigData {
		log.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = lbc.Options.ConfigData

		_, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Update(cMap)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		log.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}
