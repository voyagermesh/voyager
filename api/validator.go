package api

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

type address struct {
	RuleIndex int

	Protocol string // tcp, http
	Host     string
	PodPort  int
	NodePort int

	Paths map[string]int
}

func (a address) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.PodPort)
}

func (r Ingress) IsValid(cloudProvider string) error {
	addrs := make(map[int]*address)
	nodePorts := make(map[int]int)
	usesHTTPRule := false
	for ri, rule := range r.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true
			a := &address{RuleIndex: ri, Protocol: "http", Host: rule.Host, Paths: make(map[string]int)}

			//addr := address{Host: rule.Host}
			var err error
			a.PodPort, err = checkOptionalPort(rule.HTTP.Port)
			if err != nil {
				return fmt.Errorf("spec.rule[%d].http.port %s is invalid. Reason: %s", ri, rule.HTTP.Port, err)
			}
			if a.PodPort == 0 {
				// detect port
				if _, foundTLS := r.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoSSL {
					a.PodPort = 443
				} else {
					a.PodPort = 80
				}
			}
			if np, err := checkOptionalPort(rule.HTTP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].http.nodePort %s is invalid. Reason: %s", ri, rule.HTTP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.HTTP.NodePort)
				}
				a.NodePort = np
			}

			if ea, found := addrs[a.PodPort]; found {
				if ea.Protocol == "tcp" {
					return fmt.Errorf("spec.rule[%d].http is reusing port %d, also used in spec.rule[%d]", ri, a.PodPort, ea.RuleIndex)
				}
				if a.NodePort != ea.NodePort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %d does not match with spec.rule[%d].http.nodePort %d", ri, a.NodePort, ea.RuleIndex, ea.NodePort)
				} else {
					nodePorts[a.NodePort] = ri
				}
				a = ea // paths will be merged into the original one
			} else {
				if a.NodePort > 0 {
					if ei, found := nodePorts[a.NodePort]; found {
						return fmt.Errorf("spec.rule[%d].http is reusing nodePort %d for addr %s, also used in spec.rule[%d]", ri, a.NodePort, a, ei)
					} else {
						nodePorts[a.NodePort] = ri
					}
				}
				addrs[a.PodPort] = a
			}

			for pi, path := range rule.HTTP.Paths {
				if ei, found := a.Paths[path.Path]; found {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is reusing path %s for addr %s, also used in spec.rule[%d].http.paths[%d]", ri, pi, path, a, ri, ei)
				}
				a.Paths[path.Path] = pi

				if path.Backend.ServiceName == "" {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is missing serviceName for addr %s and path %s", ri, pi, a, path.Path)
				}
				if errs := validation.IsDNS1123Subdomain(path.Backend.ServiceName); len(errs) > 0 {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid serviceName for addr %s. Reason: %s", ri, pi, a, strings.Join(errs, ","))
				}
				if _, err := checkRequiredPort(path.Backend.ServicePort); err != nil {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is using invalid servicePort %s for addr %s and path %s. Reason: %s", ri, pi, path.Backend.ServicePort, a, path.Path, err)
				}
				for hi, hdr := range path.Backend.HeaderRule {
					if len(strings.Fields(hdr)) == 1 {
						return fmt.Errorf("spec.rule[%d].http.paths[%d].backend.headerRule[%d] is invalid for addr %s and path %s.", ri, pi, hi, a, path.Path)
					}
				}
			}
		} else if rule.TCP != nil {
			a := &address{RuleIndex: ri, Protocol: "tcp", Host: rule.Host}

			var err error
			if a.PodPort, err = checkRequiredPort(rule.TCP.Port); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.port %s is invalid. Reason: %s", ri, rule.TCP.Port, err)
			} else {
				if ea, found := addrs[a.PodPort]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing addr %s, also used in spec.rule[%d]", ri, a, ea.RuleIndex)
				}
				addrs[a.PodPort] = a
			}
			if np, err := checkOptionalPort(rule.TCP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.nodePort %s is invalid. Reason: %s", ri, rule.TCP.NodePort, err)
			} else if np > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].tcp.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.TCP.NodePort)
				}
				if ei, found := nodePorts[np]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing nodePort %d for addr %s, also used in spec.rule[%d]", ri, np, a, ei)
				} else {
					a.NodePort = np
					nodePorts[np] = ri
				}
			}

			if rule.TCP.Backend.ServiceName == "" {
				return fmt.Errorf("spec.rule[%d].tcp is missing serviceName for addr %s", ri, a)
			}
			if errs := validation.IsDNS1123Subdomain(rule.TCP.Backend.ServiceName); len(errs) > 0 {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid serviceName for addr %s. Reason: %s", ri, a, strings.Join(errs, ","))
			}
			if _, err := checkRequiredPort(rule.TCP.Backend.ServicePort); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp is using invalid servicePort %s for addr %s. Reason: %s", ri, rule.TCP.Backend.ServicePort, a, err)
			}
		} else if rule.TCP == nil && rule.HTTP == nil {
			return fmt.Errorf("spec.rule[%d] is missing both HTTP and TCP specification", ri)
		} else {
			return fmt.Errorf("spec.rule[%d] can specify either HTTP or TCP", ri)
		}
	}

	if !usesHTTPRule && r.Spec.Backend != nil {
		addrs[80] = &address{Protocol: "http", PodPort: 80}
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
	if !r.SupportsLBType(cloudProvider) {
		return fmt.Errorf("Ingress %s@%s uses unsupported LBType %s for cloud provider %s", r.Name, r.Namespace, r.LBType(), cloudProvider)
	}

	if (r.LBType() == LBTypeNodePort || r.LBType() == LBTypeHostPort) && len(r.Spec.LoadBalancerSourceRanges) > 0 {
		return fmt.Errorf("Ingress %s@%s of type %s can't use `spec.LoadBalancerSourceRanges`", r.Name, r.Namespace, r.LBType())
	}

	return nil
}

func (r Ingress) SupportsLBType(cloudProvider string) bool {
	switch r.LBType() {
	case LBTypeLoadBalancer:
		return cloudProvider == "aws" ||
			cloudProvider == "gce" ||
			cloudProvider == "gke" ||
			cloudProvider == "azure" ||
			cloudProvider == "acs" ||
			cloudProvider == "minikube"
	case LBTypeNodePort:
		return cloudProvider != "acs"
	case LBTypeHostPort:
		// TODO: https://github.com/appscode/voyager/issues/374
		return cloudProvider != "acs" && cloudProvider != "azure"
	default:
		return false
	}
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
