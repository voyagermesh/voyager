package api

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

type address struct {
	Host string
	Port int
}

type A2 struct {
	RuleIndex int

	Protocol string // tcp, http
	Host     string
	PodPort  int
	NodePort int

	Paths map[string]int
}

func (a A2) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.PodPort)
}

func (r Ingress) IsValid(cloudProvider string) error {
	addrs := make(map[int]*A2)
	nodePorts := make(map[int]int)
	usesHTTPRule := false
	for ri, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true
			a2 := &A2{RuleIndex: ri, Protocol: "http", Host: rule.Host, Paths: make(map[string]int)}

			//addr := address{Host: rule.Host}
			var err error
			a2.PodPort, err = checkOptionalPort(rule.HTTP.Port)
			if err != nil {
				return fmt.Errorf("spec.rule[%d].http.port %s is invalid. Reason: %s", ri, rule.HTTP.Port, err)
			}
			if a2.PodPort == 0 {
				// detect port
				if _, foundTLS := r.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoSSL {
					a2.PodPort = 443
				} else {
					a2.PodPort = 80
				}
			}
			if np, err := checkOptionalPort(rule.HTTP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].http.nodePort %s is invalid. Reason: %s", ri, rule.HTTP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.HTTP.NodePort)
				}
				a2.NodePort = np
			}

			if ea, found := addrs[a2.PodPort]; found {
				if ea.Protocol == "tcp" {
					return fmt.Errorf("spec.rule[%d].http is reusing port %d, also used in spec.rule[%d]", ri, a2.PodPort, ea.RuleIndex)
				}
				if a2.NodePort != ea.NodePort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %d does not match with spec.rule[%d].http.nodePort %d", ri, a2.NodePort, ea.RuleIndex, ea.NodePort)
				} else {
					nodePorts[a2.NodePort] = ri
				}
				a2 = ea // paths will be merged into the original one
			} else {
				if a2.NodePort > 0 {
					if ei, found := nodePorts[a2.NodePort]; found {
						return fmt.Errorf("spec.rule[%d].http is reusing nodePort %s for addr %s, also used in spec.rule[%d]", ri, a2.NodePort, a2, ei)
					} else {
						nodePorts[a2.NodePort] = ri
					}
				}
				addrs[a2.PodPort] = a2
			}

			for pi, path := range rule.HTTP.Paths {
				if ei, found := a2.Paths[path.Path]; found {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is reusing path %s for addr %s, also used in spec.rule[%d].http.paths[%d]", ri, pi, path, a2, ri, ei)
				}
				a2.Paths[path.Path] = pi

				if path.Backend.ServiceName == "" {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is missing serviceName for addr %s and path %s", ri, pi, a2, path.Path)
				}
				if errs := validation.IsDNS1123Subdomain(path.Backend.ServiceName); len(errs) > 0 {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid serviceName for addr %s. Reason: %s", ri, pi, a2, strings.Join(errs, ","))
				}
				if _, err := checkRequiredPort(path.Backend.ServicePort); err != nil {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid servicePort %s for addr %s and path %s. Reason: %s", ri, pi, path.Backend.ServicePort, a2, path.Path, err)
				}
				for hi, hdr := range path.Backend.HeaderRule {
					if len(strings.Fields(hdr)) == 1 {
						return fmt.Errorf("spec.rule[%d].http.paths[%d].backend.headerRule[%d] is invalid for addr %s and path %s.", ri, pi, hi, a2, path.Path)
					}
				}
			}
		} else if rule.TCP != nil {
			a2 := &A2{RuleIndex: ri, Protocol: "tcp", Host: rule.Host}

			var err error
			if a2.PodPort, err = checkRequiredPort(rule.TCP.Port); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.port %s is invalid. Reason: %s", ri, rule.TCP.Port, err)
			} else {
				if ea, found := addrs[a2.PodPort]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing addr %s, also used in spec.rule[%d]", ri, a2, ea)
				}
				addrs[a2.PodPort] = a2
			}
			if np, err := checkOptionalPort(rule.TCP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.nodePort %s is invalid. Reason: %s", ri, rule.TCP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].tcp.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.TCP.NodePort)
				}
				if ei, found := nodePorts[np]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing nodePort %s for addr %s, also used in spec.rule[%d]", ri, np, a2, ei)
				} else {
					a2.NodePort = np
					nodePorts[np] = ri
				}
			}

			if rule.TCP.Backend.ServiceName == "" {
				return fmt.Errorf("spec.rule[%d].tcp is missing serviceName for addr %s", ri, a2)
			}
			if errs := validation.IsDNS1123Subdomain(rule.TCP.Backend.ServiceName); len(errs) > 0 {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid serviceName for addr %s. Reason: %s", ri, a2, strings.Join(errs, ","))
			}
			if _, err := checkRequiredPort(rule.TCP.Backend.ServicePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid servicePort %s for addr %s. Reason: %s", ri, rule.TCP.Backend.ServicePort, a2, err)
			}
		} else if rule.TCP == nil && rule.HTTP == nil {
			return fmt.Errorf("spec.rule[%d] is missing both HTTP and TCP specification", ri)
		} else {
			return fmt.Errorf("spec.rule[%d] can specify either HTTP or TCP", ri)
		}
	}

	if !usesHTTPRule && r.Spec.Backend != nil {
		addrs[80] = &A2{Protocol: "http", PodPort: 80}
	}
	// ref: https://github.com/appscode/voyager/issues/188
	if cloudProvider == "aws" && r.LBType() == LBTypeLoadBalancer {
		if ans, ok := r.ServiceAnnotations(cloudProvider); ok {
			if v, usesAWSCertManager := ans["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; usesAWSCertManager && v != "" {
				var tp80, sp443 bool
				for svcPort, target := range addrs {
					if target.PodPort == 80 {
						tp80 = true
					}
					if svcPort == 443 {
						sp443 = true
					}
				}
				if !tp80 || sp443 {
					return fmt.Errorf("Failed to open port 443 on service for AWS cert manager for Ingress %s@%s.", r.Name, r.Namespace)
				}
			}
		}
	}
	return nil
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
