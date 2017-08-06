package ingress

import (
	"encoding/json"
	goerr "errors"
	"strconv"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/go/arrays"
	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress/template"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	"github.com/flosch/pongo2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func NewController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt config.Options,
	ingress *api.Ingress) *Controller {
	h := &Controller{
		KubeClient: kubeClient,
		ExtClient:  extClient,
		PromClient: promClient,
		Opt:        opt,
		Ingress:    ingress,
		recorder:   eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		h.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}

	h.parseOptions()
	h.parseSpec()
	return h
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

func (c *Controller) serviceEndpoints(name string, port intstr.IntOrString, hostNames []string) ([]*Endpoint, error) {
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
	service, err := c.KubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Warningln(err)
		return nil, err
	}

	if service.Spec.Type == apiv1.ServiceTypeExternalName {
		log.Infof("Found ServiceType ExternalName for service %s, Checking DNS resolver options", service.Name)
		// https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
		ep := Endpoint{
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
			c.Parsed.DNSResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
			ep.CheckHealth = resolver.CheckHealth
		}
		return []*Endpoint{&ep}, nil
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		log.Warningf("Service port %s unavailable for service %s", port.String(), service.Name)
		return nil, goerr.New("Service port unavailable")
	}
	return c.getEndpoints(service, p, hostNames)
}

func (c *Controller) getEndpoints(s *apiv1.Service, servicePort *apiv1.ServicePort, hostNames []string) (eps []*Endpoint, err error) {
	ep, err := c.KubeClient.CoreV1().Endpoints(s.Namespace).Get(s.Name, metav1.GetOptions{})
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
				if int(epPort.Port) == getTargetPort(servicePort) {
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
					ep := &Endpoint{
						Name: "server-" + epAddress.IP,
						IP:   epAddress.IP,
						Port: targetPort,
					}
					if epAddress.TargetRef != nil {
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

func (c *Controller) generateTemplate() error {
	log.Infoln("Generating Ingress template.")
	ctx, err := Context(c.Parsed)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	tpl, err := pongo2.FromString(template.HAProxyTemplate)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	r, err := tpl.Execute(ctx)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	c.ConfigData = stringutil.Fmt(r)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.Infoln("Template genareted for HAProxy")
	log.Infoln(c.ConfigData)
	return nil
}

func Context(s interface{}) (pongo2.Context, error) {
	ctx := pongo2.Context{}
	d, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return ctx, err
	}
	log.Infoln("Rendering haproxy.cfg using context", string(d))
	err = json.Unmarshal(d, &ctx)
	if err != nil {
		return ctx, errors.FromErr(err).Err()
	}
	return ctx, nil
}

func getSpecifiedPort(ports []apiv1.ServicePort, port intstr.IntOrString) (*apiv1.ServicePort, bool) {
	for _, p := range ports {
		if int(p.Port) == port.IntValue() {
			return &p, true
		}
	}
	return nil, false
}

// getTargetPort returns the numeric value of TargetPort
func getTargetPort(servicePort *apiv1.ServicePort) int {
	return servicePort.TargetPort.IntValue()
}

func (c *Controller) parseSpec() {
	log.Infoln("Parsing Engress specs for", c.Ingress.Name)
	c.Ports = make(map[int]int)
	c.Parsed.DNSResolvers = make(map[string]*api.DNSResolver)

	if c.Ingress.Spec.Backend != nil {
		log.Debugln("generating default backend", c.Ingress.Spec.Backend.RewriteRule, c.Ingress.Spec.Backend.HeaderRule)
		eps, _ := c.serviceEndpoints(c.Ingress.Spec.Backend.ServiceName, c.Ingress.Spec.Backend.ServicePort, c.Ingress.Spec.Backend.HostNames)
		c.Parsed.DefaultBackend = &Backend{
			Name:      "default-backend",
			Endpoints: eps,

			BackendRules: c.Ingress.Spec.Backend.BackendRule,
			RewriteRules: c.Ingress.Spec.Backend.RewriteRule,
			HeaderRules:  c.Ingress.Spec.Backend.HeaderRule,
		}
	}
	if len(c.Ingress.Spec.TLS) > 0 {
		c.SecretNames = make([]string, 0)
		c.HostFilter = make([]string, 0)
		for _, secret := range c.Ingress.Spec.TLS {
			c.SecretNames = append(c.SecretNames, secret.SecretName)
			c.HostFilter = append(c.HostFilter, secret.Hosts...)
		}
	}

	c.Parsed.HttpService = make([]*Service, 0)
	c.Parsed.HttpsService = make([]*Service, 0)
	c.Parsed.TCPService = make([]*TCPService, 0)

	var httpCount, httpsCount int
	for _, rule := range c.Ingress.Spec.Rules {
		host := rule.Host
		if rule.HTTP != nil {
			if ok, _ := arrays.Contains(c.HostFilter, host); ok {
				httpsCount++
			} else {
				httpCount++
			}

			for _, svc := range rule.HTTP.Paths {
				eps, _ := c.serviceEndpoints(svc.Backend.ServiceName, svc.Backend.ServicePort, svc.Backend.HostNames)
				log.Infoln("Returned service endpoints len(eps)", len(eps), "for service", svc.Backend.ServiceName)
				if len(eps) > 0 {
					def := &Service{
						Name:     "service-" + rand.Characters(6),
						Host:     host,
						AclMatch: svc.Path,
					}
					def.Backends = &Backend{
						Name:         "backend-" + rand.Characters(5),
						Endpoints:    eps,
						BackendRules: svc.Backend.BackendRule,
						RewriteRules: svc.Backend.RewriteRule,
						HeaderRules:  svc.Backend.HeaderRule,
					}
					// add the service to http or https filters.
					if ok, _ := arrays.Contains(c.HostFilter, host); ok {
						c.Parsed.HttpsService = append(c.Parsed.HttpsService, def)
					} else {
						c.Parsed.HttpService = append(c.Parsed.HttpService, def)
					}
				}
			}
		}

		// adding tcp service to the parser.
		for _, tcpSvc := range rule.TCP {
			log.Infoln(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort)

			c.Ports[tcpSvc.Port.IntValue()] = tcpSvc.Port.IntValue()
			eps, _ := c.serviceEndpoints(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort, tcpSvc.Backend.HostNames)
			log.Infoln("Returned service endpoints len(eps)", len(eps), "for service", tcpSvc.Backend.ServiceName)
			if len(eps) > 0 {
				def := &TCPService{
					Name:        "service-" + rand.Characters(6),
					Host:        host,
					Port:        tcpSvc.Port.String(),
					SecretName:  tcpSvc.SecretName,
					ALPNOptions: parseALPNOptions(tcpSvc.ALPN),
				}
				def.Backends = &Backend{
					Name:         "backend-" + rand.Characters(5),
					BackendRules: tcpSvc.Backend.BackendRule,
					Endpoints:    eps,
				}
				c.Parsed.TCPService = append(c.Parsed.TCPService, def)
				c.SecretNames = append(c.SecretNames, def.SecretName)
			}
		}
	}

	if httpCount > 0 || (c.Ingress.Spec.Backend != nil && httpsCount == 0) {
		c.Ports[80] = 80
	}

	if httpsCount > 0 {
		c.Ports[443] = 443
	}

	// ref: https://github.com/appscode/voyager/issues/188
	if c.Opt.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		if ans, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
			if v, usesAWSCertManager := ans["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; usesAWSCertManager && v != "" {
				var tp80, sp443 bool
				for svcPort, targetPort := range c.Ports {
					if targetPort == 80 {
						tp80 = true
					}
					if svcPort == 443 {
						sp443 = true
					}
				}
				if tp80 && !sp443 {
					c.Ports[443] = 80
				} else {
					log.Errorln("Failed to open port 443 on service for AWS cert manager.")
				}
			}
		}
	}
}

func (c *Controller) parseOptions() {
	if c.Ingress == nil {
		log.Infoln("Config is nil, nothing to parse")
		return
	}
	log.Infoln("Parsing annotations.")
	c.Parsed.TimeoutDefaults = c.Ingress.Timeouts()
	c.Parsed.Sticky = c.Ingress.StickySession()
	if len(c.Ingress.Spec.TLS) > 0 {
		c.Parsed.SSLCert = true
	}

	c.Parsed.Stats = c.Ingress.Stats()
	if c.Parsed.Stats {
		c.Parsed.StatsPort = c.Ingress.StatsPort()
		if name := c.Ingress.StatsSecretName(); len(name) > 0 {
			secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.ObjectMeta.Namespace).Get(name, metav1.GetOptions{})
			if err == nil {
				c.Parsed.StatsUserName = string(secret.Data["username"])
				c.Parsed.StatsPassWord = string(secret.Data["password"])
			} else {
				c.Parsed.Stats = false
				log.Errorln("Error encountered while loading Stats secret", err)
			}
		}
	}

	if c.Opt.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		c.Parsed.AcceptProxy = c.Ingress.KeepSourceIP()
	}
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
