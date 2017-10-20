package ingress

import (
	"net"
	"strings"

	"github.com/appscode/go/errors"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type UpdateMode int

const (
	UpdateStats UpdateMode = 1 << iota // Update things for stats update
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
		c.logger.Infoln("Specs have been changed updating config map data for HAProxy templates")
		cMap.Data["haproxy.cfg"] = c.HAProxyConfig

		_, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Update(cMap)
		if err != nil {
			c.recorder.Eventf(c.Ingress.ObjectReference(), apiv1.EventTypeWarning, "ConfigMapUpdateFailed", "HAProxy configuration Update failed, %s", err.Error())
			return errors.FromErr(err).Err()
		}
		// Add event only if the ConfigMap Really Updated
		c.recorder.Eventf(c.Ingress.ObjectReference(), apiv1.EventTypeNormal, "ConfigMapUpdated", "ConfigMap Updated, HAProxy will restart itself now via reloader")
		c.logger.Infoln("Config Map Updated, HAProxy will restart itself now via reloader")
	}
	return nil
}

func (c *controller) serviceRequiresUpdate(current, desired *apiv1.Service, old *api.Ingress) (*apiv1.Service, bool) {
	if current == nil {
		return nil, false // should never happen
	}

	needsUpdate := false

	// ports
	curPorts := make(map[int32]apiv1.ServicePort)
	for _, p := range current.Spec.Ports {
		curPorts[p.Port] = p
	}
	for _, dp := range desired.Spec.Ports {
		if cp, ok := curPorts[dp.Port]; !ok || // svc port not found
			cp.TargetPort.IntValue() != dp.TargetPort.IntValue() || // pod port mismatch
			(dp.NodePort != 0 && dp.NodePort != cp.NodePort) { // node port mismatch
			if dp.NodePort == 0 && cp.NodePort > 0 {
				dp.NodePort = cp.NodePort // avoid reassigning port
			}
			needsUpdate = true
		}
		delete(curPorts, dp.Port)
	}
	if len(curPorts) > 0 {
		needsUpdate = true
	}
	if needsUpdate {
		current.Spec.Ports = desired.Spec.Ports
	}

	// annotations
	if current.Annotations == nil {
		current.Annotations = make(map[string]string)
	}
	oldAnn := map[string]string{}
	if old != nil {
		if a, ok := old.ServiceAnnotations(c.Opt.CloudProvider); ok {
			oldAnn = a
		}
	}
	for k, v := range desired.Annotations {
		if cv, found := current.Annotations[k]; !found || cv != v {
			current.Annotations[k] = v
			needsUpdate = true
		}
		delete(oldAnn, k)
	}
	for k := range oldAnn {
		if _, ok := current.Annotations[k]; ok {
			delete(current.Annotations, k)
			needsUpdate = true
		}
	}

	// LoadBalancer ranges
	curRanges := sets.NewString()
	for _, ips := range current.Spec.LoadBalancerSourceRanges {
		if k, ok := ipnet(ips); ok {
			curRanges.Insert(k)
		}
	}

	desiredRanges := sets.NewString()
	for _, ips := range desired.Spec.LoadBalancerSourceRanges {
		if k, ok := ipnet(ips); ok {
			desiredRanges.Insert(k)
		}
	}
	if !curRanges.Equal(desiredRanges) {
		needsUpdate = true
		current.Spec.LoadBalancerSourceRanges = desired.Spec.LoadBalancerSourceRanges
	}

	if current.Spec.ExternalTrafficPolicy != desired.Spec.ExternalTrafficPolicy {
		needsUpdate = true
		current.Spec.ExternalTrafficPolicy = desired.Spec.ExternalTrafficPolicy
	}

	return current, needsUpdate
}

func ipnet(spec string) (string, bool) {
	spec = strings.TrimSpace(spec)
	_, ipnet, err := net.ParseCIDR(spec)
	if err != nil {
		return "", false
	}
	return ipnet.String(), true
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
