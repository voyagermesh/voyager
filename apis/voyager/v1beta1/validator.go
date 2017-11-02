package v1beta1

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
)

// +k8s:openapi-gen=false
type indices struct {
	RuleIndex int
	PathIndex int
}

// +k8s:openapi-gen=false
type Paths map[string]indices

// +k8s:openapi-gen=false
type address struct {
	Protocol       string // tcp, http
	PodPort        int
	NodePort       int
	FirstRuleIndex int
	Hosts          map[string]Paths
}

func (a address) String() string {
	for h := range a.Hosts {
		return fmt.Sprintf("%s:%d", h, a.PodPort)
	}
	return fmt.Sprintf(":%d", a.PodPort)
}

func (r *Ingress) Migrate() {
	for ti, tls := range r.Spec.TLS {
		if tls.SecretName != "" {
			r.Spec.TLS[ti].Ref = &LocalTypedReference{
				APIVersion: "v1",
				Kind:       "Secret",
				Name:       tls.SecretName,
			}
			r.Spec.TLS[ti].SecretName = ""
		}
	}
}

func (r Ingress) IsValid(cloudProvider string) error {
	for ri, rule := range r.Spec.FrontendRules {
		if _, err := checkRequiredPort(rule.Port); err != nil {
			return fmt.Errorf("spec.frontendRules[%d].port %s is invalid. Reason: %s", ri, rule.Port, err)
		}
	}
	for ti, tls := range r.Spec.TLS {
		if tls.SecretName != "" {
			return fmt.Errorf("spec.tls[%d].secretName must be migrated to spec.tls[%d].ref", ti, ti)
		} else if tls.Ref == nil {
			return fmt.Errorf("spec.tls[%d] specifies no secret name and secret ref", ti)
		} else {
			if tls.Ref.Kind != "" && !(strings.EqualFold(tls.Ref.Kind, "Secret") || strings.EqualFold(tls.Ref.Kind, "Certificate")) {
				return fmt.Errorf("spec.tls[%d].ref.kind %s is unsupported", ti, tls.Ref.Kind)
			}
			if tls.Ref.Name == "" {
				return fmt.Errorf("spec.tls[%d] specifies no secret name and secret ref name", ti)
			}
		}
	}

	addrs := make(map[int]*address)
	nodePorts := make(map[int]int)
	usesHTTPRule := false
	for ri, rule := range r.Spec.Rules {
		if rule.HTTP != nil && rule.TCP == nil {
			usesHTTPRule = true
			var err error
			var podPort, nodePort int
			podPort, err = checkOptionalPort(rule.HTTP.Port)
			if err != nil {
				return fmt.Errorf("spec.rule[%d].http.port %s is invalid. Reason: %s", ri, rule.HTTP.Port, err)
			}
			if podPort == 0 {
				// detect port
				if _, foundTLS := r.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoTLS {
					podPort = 443
				} else {
					podPort = 80
				}
			}
			if nodePort, err = checkOptionalPort(rule.HTTP.NodePort); err != nil {
				return fmt.Errorf("spec.rule[%d].http.nodePort %s is invalid. Reason: %s", ri, rule.HTTP.NodePort, err)
			} else if nodePort > 0 {
				if r.LBType() == LBTypeHostPort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %s may not be specified when `LBType` is `HostPort`", ri, rule.HTTP.NodePort)
				}
			}

			var a *address
			if ea, found := addrs[podPort]; found {
				if ea.Protocol == "tcp" {
					return fmt.Errorf("spec.rule[%d].http is reusing port %d, also used in spec.rule[%d]", ri, ea.PodPort, ea.FirstRuleIndex)
				}
				if nodePort != ea.NodePort {
					return fmt.Errorf("spec.rule[%d].http.nodePort %d does not match with nodePort %d", ri, ea.NodePort, ea.NodePort)
				} else {
					nodePorts[nodePort] = ri
				}
				a = ea // paths will be merged into the original one
			} else {
				a = &address{
					Protocol:       "http",
					PodPort:        podPort,
					NodePort:       nodePort,
					FirstRuleIndex: ri,
					Hosts:          map[string]Paths{},
				}
				if nodePort > 0 {
					if ei, found := nodePorts[nodePort]; found {
						return fmt.Errorf("spec.rule[%d].http is reusing nodePort %d for addr %s, also used in spec.rule[%d]", ri, nodePort, a, ei)
					} else {
						nodePorts[nodePort] = ri
					}
				}
				addrs[podPort] = a
			}

			for pi, path := range rule.HTTP.Paths {
				if _, found := a.Hosts[rule.Host]; !found {
					a.Hosts[rule.Host] = Paths{}
				}
				if ei, found := a.Hosts[rule.Host][path.Path]; found {
					return fmt.Errorf("spec.rule[%d].http.paths[%d] is reusing path %s for addr %s, also used in spec.rule[%d].http.paths[%d]", ri, pi, path.Path, a, ei.RuleIndex, ei)
				}
				a.Hosts[rule.Host][path.Path] = indices{RuleIndex: ri, PathIndex: pi}

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
						return fmt.Errorf("spec.rule[%d].http.paths[%d].backend.headerRule[%d] is invalid for addr %s and path %s", ri, pi, hi, a, path.Path)
					}
				}
			}
		} else if rule.TCP != nil && rule.HTTP == nil {
			var a *address
			if podPort, err := checkRequiredPort(rule.TCP.Port); err != nil {
				return fmt.Errorf("spec.rule[%d].tcp.port %s is invalid. Reason: %s", ri, rule.TCP.Port, err)
			} else {
				if ea, found := addrs[podPort]; found {
					return fmt.Errorf("spec.rule[%d].tcp is reusing addr %s, also used in spec.rule[%d]", ri, ea, ea.FirstRuleIndex)
				}
				a = &address{
					Protocol:       "tcp",
					PodPort:        podPort,
					FirstRuleIndex: ri,
					Hosts:          map[string]Paths{rule.Host: {}},
				}
				addrs[podPort] = a
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

	// If Ingress does not use any HTTP rule but defined a default backend, we need to open port 80
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
					return fmt.Errorf("failed to open port 443 on service for AWS cert manager for Ingress %s@%s", r.Name, r.Namespace)
				}
			}
		}
	}
	if !r.SupportsLBType(cloudProvider) {
		return fmt.Errorf("ingress %s@%s uses unsupported LBType %s for cloud provider %s", r.Name, r.Namespace, r.LBType(), cloudProvider)
	}

	if (r.LBType() == LBTypeNodePort || r.LBType() == LBTypeHostPort || r.LBType() == LBTypeInternal) && len(r.Spec.LoadBalancerSourceRanges) > 0 {
		return fmt.Errorf("ingress %s@%s of type %s can't use `spec.LoadBalancerSourceRanges`", r.Name, r.Namespace, r.LBType())
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
			cloudProvider == "openstack" ||
			cloudProvider == "minikube"
	case LBTypeNodePort:
		return cloudProvider != "acs"
	case LBTypeHostPort:
		// TODO: https://github.com/appscode/voyager/issues/374
		return cloudProvider != "acs" && cloudProvider != "azure" && cloudProvider != "gce" && cloudProvider != "gke"
	case LBTypeInternal:
		return true
	default:
		return false
	}
}

func checkRequiredPort(port intstr.IntOrString) (int, error) {
	if port.Type == intstr.Int {
		if port.IntVal <= 0 {
			return 0, fmt.Errorf("port %s must a +ve interger", port)
		}
		return int(port.IntVal), nil
	} else if port.Type == intstr.String {
		return strconv.Atoi(port.StrVal)
	}
	return 0, fmt.Errorf("invalid data type %v for port %s", port.Type, port)
}

func checkOptionalPort(port intstr.IntOrString) (int, error) {
	if port.Type == intstr.Int {
		if port.IntVal < 0 {
			return 0, fmt.Errorf("port %s can't be -ve interger", port)
		}
		return int(port.IntVal), nil
	} else if port.Type == intstr.String {
		if port.StrVal == "" {
			return 0, nil
		}
		return strconv.Atoi(port.StrVal)
	}
	return 0, fmt.Errorf("invalid data type %v for port %s", port.Type, port)
}

func (c Certificate) IsValid(cloudProvider string) error {
	if len(c.Spec.Domains) == 0 {
		return fmt.Errorf("doamin list is empty")
	}

	if c.Spec.ChallengeProvider.HTTP == nil && c.Spec.ChallengeProvider.DNS == nil {
		return fmt.Errorf("certificate has no valid challange provider")
	}

	if c.Spec.ChallengeProvider.HTTP != nil && c.Spec.ChallengeProvider.DNS != nil {
		return fmt.Errorf("invalid provider specification, used both http and dns provider")
	}

	if c.Spec.ChallengeProvider.HTTP != nil {
		if len(c.Spec.ChallengeProvider.HTTP.Ingress.Name) == 0 ||
			(c.Spec.ChallengeProvider.HTTP.Ingress.APIVersion != SchemeGroupVersion.String() && c.Spec.ChallengeProvider.HTTP.Ingress.APIVersion != "extensions/v1beta1") {
			return fmt.Errorf("invalid ingress reference")
		}
	}

	if c.Spec.ChallengeProvider.DNS != nil {
		if len(c.Spec.ChallengeProvider.DNS.Provider) == 0 {
			return fmt.Errorf("no dns provider name specified")
		}
		if c.Spec.ChallengeProvider.DNS.CredentialSecretName == "" {
			useCredentialFromEnv := (cloudProvider == "aws" && c.Spec.ChallengeProvider.DNS.Provider == "aws" && c.Spec.ChallengeProvider.DNS.CredentialSecretName == "") ||
				(sets.NewString("gce", "gke").Has(cloudProvider) && sets.NewString("googlecloud", "gcloud", "gce", "gke").Has(c.Spec.ChallengeProvider.DNS.Provider) && c.Spec.ChallengeProvider.DNS.CredentialSecretName == "")
			if !useCredentialFromEnv {
				return fmt.Errorf("missing dns challenge provider credential")
			}
		}
	}

	if len(c.Spec.ACMEUserSecretName) == 0 {
		return fmt.Errorf("no user secret name specified")
	}

	if c.Spec.Storage.Secret != nil && c.Spec.Storage.Vault != nil {
		return fmt.Errorf("invalid storage specification, used both storage")
	}

	return nil
}
