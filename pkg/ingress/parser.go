package ingress

import (
	goerr "errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/haproxy"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func NewController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	services core.ServiceLister,
	endpoints core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) *Controller {
	ctrl := &Controller{
		KubeClient:      kubeClient,
		ExtClient:       extClient,
		PromClient:      promClient,
		ServiceLister:   services,
		EndpointsLister: endpoints,
		Opt:             opt,
		Ingress:         ingress,
		recorder:        eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		ctrl.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return ctrl
}

func (c *Controller) SupportsLBType() bool {
	switch c.Ingress.LBType() {
	case api.LBTypeLoadBalancer:
		return c.Opt.CloudProvider == "aws" ||
			c.Opt.CloudProvider == "gce" ||
			c.Opt.CloudProvider == "gke" ||
			c.Opt.CloudProvider == "azure" ||
			c.Opt.CloudProvider == "acs" ||
			c.Opt.CloudProvider == "minikube"
	case api.LBTypeNodePort, api.LBTypeHostPort:
		return c.Opt.CloudProvider != "acs"
	default:
		return false
	}
}

func (c *Controller) serviceEndpoints(parsed *haproxy.TemplateData, name string, port intstr.IntOrString, hostNames []string) ([]*haproxy.Endpoint, error) {
	log.Infoln("getting endpoints for ", c.Ingress.Namespace, name, "port", port)

	// the following lines giving support to
	// serviceName.namespaceName or serviceName in the same namespace to the
	// ingress
	var namespace string = c.Ingress.Namespace
	if strings.Contains(name, ".") {
		namespace = name[strings.Index(name, ".")+1:]
		name = name[:strings.Index(name, ".")]
	}

	log.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := c.ServiceLister.Services(namespace).Get(name)
	if err != nil {
		log.Warningln(err)
		return nil, err
	}

	if service.Spec.Type == apiv1.ServiceTypeExternalName {
		log.Infof("Found ServiceType ExternalName for service %s, Checking DNS resolver options", service.Name)
		// https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
		ep := haproxy.Endpoint{
			Name:         "external",
			Port:         port.String(),
			ExternalName: service.Spec.ExternalName,
		}

		var resolver *api.DNSResolver
		var err error
		ep.UseDNSResolver, resolver, err = api.DNSResolverForService(*service)
		if err != nil {
			return nil, errors.FromErr(err).Err()
		}
		if ep.UseDNSResolver && resolver != nil {
			parsed.DNSResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
			ep.CheckHealth = resolver.CheckHealth
		}
		return []*haproxy.Endpoint{&ep}, nil
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		log.Warningf("Service port %s unavailable for service %s", port.String(), service.Name)
		return nil, goerr.New("Service port unavailable")
	}
	return c.getEndpoints(service, p, hostNames)
}

func (c *Controller) getEndpoints(s *apiv1.Service, servicePort *apiv1.ServicePort, hostNames []string) (eps []*haproxy.Endpoint, err error) {
	ep, err := c.EndpointsLister.Endpoints(s.Namespace).Get(s.Name)
	if err != nil {
		log.Warningln(err)
		return nil, err
	}

	// The intent here is to create a union of all subsets that match a targetPort.
	// We know the endpoint already matches the service, so all pod ips that have
	// the target port are capable of service traffic for it.
	for _, ss := range ep.Subsets {
		for _, epPort := range ss.Ports {
			var targetPort string
			switch servicePort.TargetPort.Type {
			case intstr.Int:
				if int(epPort.Port) == servicePort.TargetPort.IntValue() {
					targetPort = servicePort.TargetPort.String()
				}
			case intstr.String:
				if epPort.Name == servicePort.TargetPort.StrVal {
					targetPort = strconv.Itoa(int(epPort.Port))
				} else if epPort.Name == servicePort.Name {
					targetPort = strconv.Itoa(int(epPort.Port))
				}
			}
			if targetPort == "" {
				if len(ss.Ports) == 1 {
					targetPort = strconv.Itoa(int(epPort.Port))
				} else {
					log.Debugf("Target port %s empty for service %s. skipping.", servicePort.String(), s.Name)
					continue
				}
			}

			log.Infof("Found target port %s for service %s", targetPort, s.Name)
			for _, epAddress := range ss.Addresses {
				if isForwardable(hostNames, epAddress.Hostname) {
					ep := &haproxy.Endpoint{
						Name: "pod-" + epAddress.IP,
						IP:   epAddress.IP,
						Port: targetPort,
					}
					if epAddress.TargetRef != nil {
						// Use PodList via service selector
						pod, err := c.KubeClient.CoreV1().Pods(epAddress.TargetRef.Namespace).Get(epAddress.TargetRef.Name, metav1.GetOptions{})
						if err != nil {
							log.Errorln("Error getting endpoint pod", err)
						} else {
							if pod.Annotations != nil {
								if val, ok := pod.Annotations[api.BackendWeight]; ok {
									ep.Weight, _ = strconv.Atoi(val)
								}
							}
						}
					}
					eps = append(eps, ep)
				}
			}
		}
	}
	return
}

func isForwardable(hostNames []string, hostName string) bool {
	if len(hostNames) <= 0 {
		return true
	}

	for _, name := range hostNames {
		if strings.EqualFold(name, hostName) {
			return true
		}
	}
	return false
}

func getSpecifiedPort(ports []apiv1.ServicePort, port intstr.IntOrString) (*apiv1.ServicePort, bool) {
	for _, p := range ports {
		if int(p.Port) == port.IntValue() {
			return &p, true
		}
	}
	return nil, false
}

func (c *Controller) generateConfig() error {
	var parsed haproxy.TemplateData

	var si haproxy.SharedInfo
	si.Sticky = c.Ingress.StickySession()
	if c.Opt.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		si.AcceptProxy = c.Ingress.KeepSourceIP()
	}
	parsed.SharedInfo = si
	parsed.TimeoutDefaults = c.Ingress.Timeouts()

	if c.Ingress.Stats() {
		stats := &haproxy.StatsInfo{}
		stats.Port = c.Ingress.StatsPort()
		if name := c.Ingress.StatsSecretName(); len(name) > 0 {
			secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.ObjectMeta.Namespace).Get(name, metav1.GetOptions{})
			if err == nil {
				stats.Username = string(secret.Data["username"])
				stats.PassWord = string(secret.Data["password"])
			} else {
				return fmt.Errorf("Failed to load stats secret for ingress %s@%s", c.Ingress.Name, c.Ingress.Namespace)
			}
		}
		parsed.Stats = stats
	}
	parsed.DNSResolvers = make(map[string]*api.DNSResolver)

	if c.Ingress.Spec.Backend != nil {
		eps, err := c.serviceEndpoints(&parsed, c.Ingress.Spec.Backend.ServiceName, c.Ingress.Spec.Backend.ServicePort, c.Ingress.Spec.Backend.HostNames)
		if err != nil {
			return err
		}
		parsed.DefaultBackend = &haproxy.Backend{
			Endpoints:    eps,
			BackendRules: c.Ingress.Spec.Backend.BackendRule,
			RewriteRules: c.Ingress.Spec.Backend.RewriteRule,
			HeaderRules:  c.Ingress.Spec.Backend.HeaderRule,
		}
	}

	parsed.HTTPService = make([]*haproxy.HTTPService, 0)
	parsed.TCPService = make([]*haproxy.TCPService, 0)

	type httpKey struct {
		Port    int
		UsesSSL bool
	}
	httpServices := make(map[httpKey][]*haproxy.HTTPPath)
	for _, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			httpPaths := make([]*haproxy.HTTPPath, 0)
			for _, path := range rule.HTTP.Paths {
				eps, err := c.serviceEndpoints(&parsed, path.Backend.ServiceName, path.Backend.ServicePort, path.Backend.HostNames)
				if err != nil {
					return err
				}
				if len(eps) > 0 {
					httpPaths = append(httpPaths, &haproxy.HTTPPath{
						Host: rule.Host,
						Path: path.Path,
						Backend: haproxy.Backend{
							Name:         "service-" + rand.Characters(6),
							Endpoints:    eps,
							BackendRules: path.Backend.BackendRule,
							RewriteRules: path.Backend.RewriteRule,
							HeaderRules:  path.Backend.HeaderRule,
						},
					})
				}
			}

			var key httpKey
			if _, foundTLS := c.Ingress.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoSSL {
				key.UsesSSL = true
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					key.Port = port
				} else {
					key.Port = 443
				}
			} else {
				key.UsesSSL = false
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					key.Port = port
				} else {
					key.Port = 80
				}
			}

			if v, ok := httpServices[key]; ok {
				httpServices[key] = append(v, httpPaths...)
			} else {
				httpServices[key] = httpPaths
			}
		} else if rule.TCP != nil {
			eps, err := c.serviceEndpoints(&parsed, rule.TCP.Backend.ServiceName, rule.TCP.Backend.ServicePort, rule.TCP.Backend.HostNames)
			if err != nil {
				return err
			}
			if len(eps) > 0 {
				def := &haproxy.TCPService{
					SharedInfo:   si,
					FrontendName: "tcp-" + rule.TCP.Port.String(),
					Host:         rule.Host,
					Port:         rule.TCP.Port.String(),
					ALPNOptions:  parseALPNOptions(rule.TCP.ALPN),
					Backend: haproxy.Backend{
						BackendRules: rule.TCP.Backend.BackendRule,
						Endpoints:    eps,
					},
				}
				if secretName, ok := c.Ingress.FindTLSSecret(rule.Host); ok && !rule.TCP.NoSSL {
					def.SecretName = secretName
				}
				parsed.TCPService = append(parsed.TCPService, def)
			}
		}
	}

	for key := range httpServices {
		value := httpServices[key]
		parsed.HTTPService = append(parsed.HTTPService, &haproxy.HTTPService{
			SharedInfo:   si,
			FrontendName: "fix-it",
			Port:         key.Port,
			UsesSSL:      key.UsesSSL,
			Paths:        value,
		})
	}

	var err error
	c.HAProxyConfig, err = haproxy.RenderConfig(parsed)
	return err
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
