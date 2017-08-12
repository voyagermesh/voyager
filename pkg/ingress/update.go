package ingress

import (
	"reflect"
	"strconv"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (c *controller) updateConfigMap() error {
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

func (c *controller) updateLBSvc() (*apiv1.Service, error) {
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	curPorts := make(map[int32]apiv1.ServicePort)
	for _, p := range svc.Spec.Ports {
		curPorts[p.Port] = p
	}
	svc.Spec.Ports = make([]apiv1.ServicePort, 0)

	mappings, err := c.Ingress.PortMappings(c.Opt.CloudProvider)
	if err != nil {
		return nil, err
	}
	for svcPort, target := range mappings {
		if sp, found := curPorts[int32(svcPort)]; found && sp.TargetPort.IntValue() == target.PodPort {
			if target.NodePort > 0 {
				sp.NodePort = int32(target.NodePort) // ensure preferred NodePort is used.
			}
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

	return c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(svc)
}

func (c *controller) updateServiceAnnotations(old *api.Ingress, new *api.Ingress) error {
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
