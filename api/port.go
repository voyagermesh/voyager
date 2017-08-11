package api

import (
	"fmt"
)

// PortMappings contains a map of Service Port to HAProxy port (svc.Port -> {svc.TargetPort, svc.NodePort}).
// HAProxy pods binds to the target ports. Service ports are used to open loadbalancer/firewall.
// Usually target port == service port with one exception for LoadBalancer type service in AWS.
// If AWS cert manager is used then a 443 -> 80 port mapping is added.
func (r Ingress) PortMappings(cloudProvider string) (map[int]int, error) {
	mappings := make(map[int]int)

	usesHTTPRule := false
	for _, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true
			if _, foundTLS := r.FindTLSSecret(rule.Host); foundTLS {
				mappings[443] = 443
			} else {
				mappings[80] = 80
			}
		}
		if rule.TCP != nil {
			for _, port := range rule.TCP {
				p := port.Port.IntValue()
				if p > 0 {
					mappings[p] = p
				}
			}
		}
	}

	if !usesHTTPRule && r.Spec.Backend != nil {
		mappings[80] = 80
	}
	// ref: https://github.com/appscode/voyager/issues/188
	if cloudProvider == "aws" && r.LBType() == LBTypeLoadBalancer {
		if ans, ok := r.ServiceAnnotations(cloudProvider); ok {
			if v, usesAWSCertManager := ans["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; usesAWSCertManager && v != "" {
				var tp80, sp443 bool
				for svcPort, target := range mappings {
					if target == 80 {
						tp80 = true
					}
					if svcPort == 443 {
						sp443 = true
					}
				}
				if tp80 && !sp443 {
					mappings[443] = 80
				} else {
					return nil, fmt.Errorf("Failed to open port 443 on service for AWS cert manager for Ingress %s@%s.", r.Name, r.Namespace)
				}
			}
		}
	}
	return mappings, nil
}
