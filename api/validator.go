package api

import (
	"fmt"
)

func (r Ingress) IsValid() error {
	type Address struct {
		Host string
		Port int
	}
	addrs := make(map[Address]int)
	for ri, rule := range r.Spec.Rules {
		if rule.Host == "" {
			return fmt.Errorf("spec.rule[%d] is missing Host", ri)
		}
		if rule.TCP == nil && rule.HTTP == nil {
			return fmt.Errorf("spec.rule[%d] is missing both HTTP and TCP specification", ri)
		}
		if rule.TCP != nil {
			port := rule.TCP.Port.IntValue()
			if port <= 0 {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid port %s", ri, rule.TCP.Port)
			}
			addr := Address{Host: rule.Host, Port: port}
			if ei, found := addrs[addr]; found {
				return fmt.Errorf("spec.rule[%d].tcp is reusing %s:%d, also used in spec.rule[%d]", ri, rule.Host, port, ei)
			}
			addrs[addr] = ri

			if rule.TCP.Backend.ServiceName == "" {
				return fmt.Errorf("spec.rule[%d].tcp is missing serviceName for addr %s:%d", ri, rule.Host, port)
			}

			svcPort := rule.TCP.Backend.ServicePort.IntValue()
			if svcPort <= 0 {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid servicePort %s for addr %s:%d", ri, rule.TCP.Backend.ServicePort, rule.Host, port)
			}

			if len(rule.TCP.Backend.HostNames) > 0 && rule.TCP.Backend.HostExpander != "" {
				return fmt.Errorf("spec.rule[%d].tcp is using both hostNames and hostExpander for addr %s:%d", ri, rule.Host, port)
			}
		}
		if rule.HTTP != nil {
			port := rule.HTTP.Port.IntValue()
			if port <= 0 {
				return fmt.Errorf("Rule #%d is using invalid port %s for HTTP ", ri, rule.HTTP.Port)
			}
			addr := Address{Host: rule.Host, Port: port}
			if ei, found := addrs[addr]; found {
				return fmt.Errorf("Rule #%d is reusing %s:%d for HTTP, also used in rule #%d", ri, rule.Host, rule.HTTP.Port, ei)
			}
			addrs[addr] = ri

			paths := make(map[string]int)
			for pi, path := range rule.HTTP.Paths {
				if ei, found := paths[path.Path]; found {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is reusing path %s for addr %s:%d, also used in spec.rule[%d].http.paths[%d]", ri, pi, path, rule.Host, port, ri, ei)
				}
				paths[path.Path] = pi

				if path.Backend.ServiceName == "" {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is missing serviceName for addr %s:%d and path %s", ri, pi, rule.Host, port, path.Path)
				}
				svcPort := path.Backend.ServicePort.IntValue()
				if svcPort <= 0 {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid servicePort %s for addr %s:%d and path %s", ri, pi, path.Backend.ServicePort, rule.Host, port, path.Path)
				}
				if len(path.Backend.HostNames) > 0 && path.Backend.HostExpander != "" {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using both hostNames and hostExpander for addr %s:%d and path %s", ri, pi, rule.Host, port, path.Path)
				}
			}
		}
	}
	return nil
}
