package api

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Target struct {
	PodPort  int
	NodePort int
}

// PortMappings contains a map of Service Port to HAProxy port (svc.Port -> {svc.TargetPort, svc.NodePort}).
// HAProxy pods binds to the target ports. Service ports are used to open loadbalancer/firewall.
// Usually target port == service port with one exception for LoadBalancer type service in AWS.
// If AWS cert manager is used then a 443 -> 80 port mapping is added.
func (r Ingress) PortMappings(cloudProvider string) (map[int]Target, error) {
	mappings := make(map[int]Target)

	usesHTTPRule := false
	for _, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true
			if _, foundTLS := r.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoSSL {
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					mappings[port] = Target{PodPort: port, NodePort: rule.HTTP.NodePort.IntValue()}
				} else {
					mappings[443] = Target{PodPort: 443, NodePort: rule.HTTP.NodePort.IntValue()}
				}
			} else {
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					mappings[port] = Target{PodPort: port, NodePort: rule.HTTP.NodePort.IntValue()}
				} else {
					mappings[80] = Target{PodPort: 80, NodePort: rule.HTTP.NodePort.IntValue()}
				}
			}
		} else if rule.TCP != nil {
			mappings[rule.TCP.Port.IntValue()] = Target{
				PodPort:  rule.TCP.Port.IntValue(),
				NodePort: rule.TCP.NodePort.IntValue(),
			}
		}
	}

	if !usesHTTPRule && r.Spec.Backend != nil {
		mappings[80] = Target{PodPort: 80}
	}
	// ref: https://github.com/appscode/voyager/issues/188
	if cloudProvider == "aws" && r.LBType() == LBTypeLoadBalancer {
		if ans, ok := r.ServiceAnnotations(cloudProvider); ok {
			if v, usesAWSCertManager := ans["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; usesAWSCertManager && v != "" {
				var tp80, sp443 bool
				for svcPort, target := range mappings {
					if target.PodPort == 80 {
						tp80 = true
					}
					if svcPort == 443 {
						sp443 = true
					}
				}
				if tp80 && !sp443 {
					mappings[443] = Target{PodPort: 80}
				} else {
					return nil, fmt.Errorf("Failed to open port 443 on service for AWS cert manager for Ingress %s@%s.", r.Name, r.Namespace)
				}
			}
		}
	}
	return mappings, nil
}

func (r Ingress) PodPorts() []int {
	ports := sets.NewInt()
	usesHTTPRule := false
	for _, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true
			if port := rule.HTTP.Port.IntValue(); port > 0 {
				ports.Insert(port)
			} else {
				if _, ok := r.FindTLSSecret(rule.Host); ok && !rule.HTTP.NoSSL {
					ports.Insert(443)
				} else {
					ports.Insert(80)
				}
			}
		} else if rule.TCP != nil {
			if port := rule.TCP.Port.IntValue(); port > 0 {
				ports.Insert(port)
			}
		}
	}
	if !usesHTTPRule && r.Spec.Backend != nil {
		ports.Insert(80)
	}
	return ports.List()
}
