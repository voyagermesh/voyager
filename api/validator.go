package api

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

type address struct {
	Host string
	Port int
}

func (a address) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func (r Ingress) IsValid(cloudProvider string) error {
	addrs := make(map[address]int)
	nodePorts := make(map[int]int)
	for ri, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			addr := address{Host: rule.Host}
			if port, err := checkRequiredPort(rule.HTTP.Port); err != nil {
				return fmt.Errorf("spec.rule[%d].http.port %s is invalid. Reason: %s", ri, rule.HTTP.Port, err)
			} else {
				addr.Port = port
			}
			if ei, found := addrs[addr]; found {
				return fmt.Errorf("spec.rule[%d].http is reusing addr %s, also used in spec.rule[%d]", ri, addr, ei)
			}
			addrs[addr] = ri

			if np, err := checkOptionalPort(rule.HTTP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].http.nodePort %s is invalid. Reason: %s", ri, rule.HTTP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.HTTP.NodePort)
				}
				if ei, found := nodePorts[np]; found {
					return fmt.Errorf("spec.rule[%d].http is reusing nodePort %s for addr %s, also used in spec.rule[%d]", ri, np, addr, ei)
				} else {
					nodePorts[np] = ri
				}
			}

			paths := make(map[string]int)
			for pi, path := range rule.HTTP.Paths {
				if ei, found := paths[path.Path]; found {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is reusing path %s for addr %s, also used in spec.rule[%d].http.paths[%d]", ri, pi, path, addr, ri, ei)
				}
				paths[path.Path] = pi

				if path.Backend.ServiceName == "" {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is missing serviceName for addr %s and path %s", ri, pi, addr, path.Path)
				}
				if _, err := checkRequiredPort(path.Backend.ServicePort); err != nil {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid servicePort %s for addr %s and path %s. Reason: %s", ri, pi, path.Backend.ServicePort, addr, path.Path, err)
				}

				for hi, hdr := range path.Backend.HeaderRule {
					if len(strings.Fields(hdr)) == 1 {
						return fmt.Errorf("spec.rule[%d].http.paths[%d].backend.headerRule[%d] is invalid for addr %s and path %s.", ri, pi, hi, addr, path.Path)
					}
				}

			}
		} else if rule.TCP != nil {
			addr := address{Host: rule.Host}
			if port, err := checkRequiredPort(rule.TCP.Port); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.port %s is invalid. Reason: %s", ri, rule.TCP.Port, err)
			} else {
				addr.Port = port
			}
			if ei, found := addrs[addr]; found {
				return fmt.Errorf("spec.rule[%d].tcp is reusing addr %s, also used in spec.rule[%d]", ri, addr, ei)
			}
			addrs[addr] = ri

			if np, err := checkOptionalPort(rule.TCP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.nodePort %s is invalid. Reason: %s", ri, rule.TCP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].tcp.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.TCP.NodePort)
				}
				if ei, found := nodePorts[np]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing nodePort %s for addr %s, also used in spec.rule[%d]", ri, np, addr, ei)
				} else {
					nodePorts[np] = ri
				}
			}

			if rule.TCP.Backend.ServiceName == "" {
				return fmt.Errorf("spec.rule[%d].tcp is missing serviceName for addr %s", ri, addr)
			}
			if _, err := checkRequiredPort(rule.TCP.Backend.ServicePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid servicePort %s for addr %s. Reason: %s", ri, rule.TCP.Backend.ServicePort, addr, err)
			}
			if len(rule.TCP.Backend.HeaderRule) > 0 {
				return fmt.Errorf("spec.rule[%d].tcp.backend.headerRule must be empty for addr %s", ri, addr)
			}
			if len(rule.TCP.Backend.RewriteRule) > 0 {
				return fmt.Errorf("spec.rule[%d].tcp.backend.rewriteRule must be empty for addr %s", ri, addr)
			}
		} else if rule.TCP == nil && rule.HTTP == nil {
			return fmt.Errorf("spec.rule[%d] is missing both HTTP and TCP specification", ri)
		} else {
			return fmt.Errorf("spec.rule[%d] can specify either HTTP or TCP", ri)
		}
	}

	_, err := r.PortMappings(cloudProvider)
	return err
}

func checkRequiredPort(port intstr.IntOrString) (int, error) {
	if port.Type == intstr.Int {
		if port.IntVal <= 0 {
			return 0, fmt.Errorf("Port %s must a +ve interger", port)
		}
		return int(port.IntVal), nil
	} else if port.Type == intstr.String {
		return strconv.Atoi(port.StrVal)
	}
	return 0, fmt.Errorf("Invalid data type %v for port %s", port.Type, port)
}

func checkOptionalPort(port intstr.IntOrString) (int, error) {
	if port.Type == intstr.Int {
		if port.IntVal < 0 {
			return 0, fmt.Errorf("Port %s can't be -ve interger", port)
		}
		return int(port.IntVal), nil
	} else if port.Type == intstr.String {
		if port.StrVal == "" {
			return 0, nil
		}
		return strconv.Atoi(port.StrVal)
	}
	return 0, fmt.Errorf("Invalid data type %v for port %s", port.Type, port)
}
