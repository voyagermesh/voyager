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

func (lbc *Controller) SupportsLBType() bool {
	switch lbc.Ingress.LBType() {
	case api.LBTypeLoadBalancer:
		return lbc.Opt.CloudProvider == "aws" ||
			lbc.Opt.CloudProvider == "gce" ||
			lbc.Opt.CloudProvider == "gke" ||
			lbc.Opt.CloudProvider == "azure" ||
			lbc.Opt.CloudProvider == "acs" ||
			lbc.Opt.CloudProvider == "minikube"
	case api.LBTypeNodePort, api.LBTypeHostPort:
		return lbc.Opt.CloudProvider != "acs"
	default:
		return false
	}
}

func (lbc *Controller) serviceEndpoints(name string, port intstr.IntOrString, hostNames []string) ([]*Endpoint, error) {
	log.Infoln("getting endpoints for ", lbc.Ingress.Namespace, name, "port", port)

	// the following lines giving support to
	// serviceName.namespaceName or serviceName in the same namespace to the
	// ingress
	var namespace string = lbc.Ingress.Namespace
	if strings.Contains(name, ".") {
		namespace = name[strings.Index(name, ".")+1:]
		name = name[:strings.Index(name, ".")]
	}

	log.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := lbc.KubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Warningln(err)
		return nil, err
	}

	if service.Spec.Type == apiv1.ServiceTypeExternalName {
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
			return nil, err
		}
		if ep.UseDNSResolver && resolver != nil {
			lbc.Parsed.DNSResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
			ep.CheckHealth = resolver.CheckHealth
		}
		return []*Endpoint{&ep}, nil
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		log.Warningln("service port unavailable")
		return nil, goerr.New("service port unavailable")
	}
	return lbc.getEndpoints(service, p, hostNames)
}

func (lbc *Controller) getEndpoints(s *apiv1.Service, servicePort *apiv1.ServicePort, hostNames []string) (eps []*Endpoint, err error) {
	ep, err := lbc.KubeClient.CoreV1().Endpoints(s.Namespace).Get(s.Name, metav1.GetOptions{})
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
					log.Debugln("targert port empty. skipping.")
					continue
				}
			}

			log.Infoln("targert port", targetPort)
			for _, epAddress := range ss.Addresses {
				if isForwardable(hostNames, epAddress.Hostname) {
					ep := &Endpoint{
						Name: "server-" + epAddress.IP,
						IP:   epAddress.IP,
						Port: targetPort,
					}
					if epAddress.TargetRef != nil {
						pod, err := lbc.KubeClient.CoreV1().Pods(epAddress.TargetRef.Namespace).Get(epAddress.TargetRef.Name, metav1.GetOptions{})
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

func (lbc *Controller) generateTemplate() error {
	log.Infoln("Generating Ingress template.")
	ctx, err := Context(lbc.Parsed)
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

	lbc.ConfigData = stringutil.Fmt(r)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.Infoln("Template genareted for HAProxy")
	log.Infoln(lbc.ConfigData)
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

func (lbc *Controller) parseSpec() {
	log.Infoln("Parsing Engress specs")
	lbc.Ports = make(map[int]int)
	lbc.Parsed.DNSResolvers = make(map[string]*api.DNSResolver)

	if lbc.Ingress.Spec.Backend != nil {
		log.Debugln("generating default backend", lbc.Ingress.Spec.Backend.RewriteRule, lbc.Ingress.Spec.Backend.HeaderRule)
		eps, _ := lbc.serviceEndpoints(lbc.Ingress.Spec.Backend.ServiceName, lbc.Ingress.Spec.Backend.ServicePort, lbc.Ingress.Spec.Backend.HostNames)
		lbc.Parsed.DefaultBackend = &Backend{
			Name:      "default-backend",
			Endpoints: eps,

			BackendRules: lbc.Ingress.Spec.Backend.BackendRule,
			RewriteRules: lbc.Ingress.Spec.Backend.RewriteRule,
			HeaderRules:  lbc.Ingress.Spec.Backend.HeaderRule,
		}
	}
	if len(lbc.Ingress.Spec.TLS) > 0 {
		lbc.SecretNames = make([]string, 0)
		lbc.HostFilter = make([]string, 0)
		for _, secret := range lbc.Ingress.Spec.TLS {
			lbc.SecretNames = append(lbc.SecretNames, secret.SecretName)
			lbc.HostFilter = append(lbc.HostFilter, secret.Hosts...)
		}
	}

	lbc.Parsed.HttpService = make([]*Service, 0)
	lbc.Parsed.HttpsService = make([]*Service, 0)
	lbc.Parsed.TCPService = make([]*TCPService, 0)

	var httpCount, httpsCount int
	for _, rule := range lbc.Ingress.Spec.Rules {
		host := rule.Host
		if rule.HTTP != nil {
			if ok, _ := arrays.Contains(lbc.HostFilter, host); ok {
				httpsCount++
			} else {
				httpCount++
			}

			for _, svc := range rule.HTTP.Paths {
				eps, _ := lbc.serviceEndpoints(svc.Backend.ServiceName, svc.Backend.ServicePort, svc.Backend.HostNames)
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
					if ok, _ := arrays.Contains(lbc.HostFilter, host); ok {
						lbc.Parsed.HttpsService = append(lbc.Parsed.HttpsService, def)
					} else {
						lbc.Parsed.HttpService = append(lbc.Parsed.HttpService, def)
					}
				}
			}
		}

		// adding tcp service to the parser.
		for _, tcpSvc := range rule.TCP {
			log.Infoln(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort)

			lbc.Ports[tcpSvc.Port.IntValue()] = tcpSvc.Port.IntValue()
			eps, _ := lbc.serviceEndpoints(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort, tcpSvc.Backend.HostNames)
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
				lbc.Parsed.TCPService = append(lbc.Parsed.TCPService, def)
				lbc.SecretNames = append(lbc.SecretNames, def.SecretName)
			}
		}
	}

	if httpCount > 0 || (lbc.Ingress.Spec.Backend != nil && httpsCount == 0) {
		lbc.Ports[80] = 80
	}

	if httpsCount > 0 {
		lbc.Ports[443] = 443
	}

	// ref: https://github.com/appscode/voyager/issues/188
	if lbc.Opt.CloudProvider == "aws" && lbc.Ingress.LBType() == api.LBTypeLoadBalancer {
		if ans, ok := lbc.Ingress.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
			if v, usesAWSCertManager := ans["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; usesAWSCertManager && v != "" {
				var tp80, sp443 bool
				for svcPort, targetPort := range lbc.Ports {
					if targetPort == 80 {
						tp80 = true
					}
					if svcPort == 443 {
						sp443 = true
					}
				}
				if tp80 && !sp443 {
					lbc.Ports[443] = 80
				} else {
					log.Errorln("Failed to open port 443 on service for AWS cert manager.")
				}
			}
		}
	}
}

func (lbc *Controller) parseOptions() {
	if lbc.Ingress == nil {
		log.Infoln("Config is nil, nothing to parse")
		return
	}
	log.Infoln("Parsing annotations.")
	lbc.Parsed.TimeoutDefaults = lbc.Ingress.Timeouts()
	lbc.Parsed.Sticky = lbc.Ingress.StickySession()
	if len(lbc.Ingress.Spec.TLS) > 0 {
		lbc.Parsed.SSLCert = true
	}

	lbc.Parsed.Stats = lbc.Ingress.Stats()
	if lbc.Parsed.Stats {
		lbc.Parsed.StatsPort = lbc.Ingress.StatsPort()
		if name := lbc.Ingress.StatsSecretName(); len(name) > 0 {
			secret, err := lbc.KubeClient.CoreV1().Secrets(lbc.Ingress.ObjectMeta.Namespace).Get(name, metav1.GetOptions{})
			if err == nil {
				lbc.Parsed.StatsUserName = string(secret.Data["username"])
				lbc.Parsed.StatsPassWord = string(secret.Data["password"])
			} else {
				lbc.Parsed.Stats = false
				log.Errorln("Error encountered while loading Stats secret", err)
			}
		}
	}

	if lbc.Opt.CloudProvider == "aws" && lbc.Ingress.LBType() == api.LBTypeLoadBalancer {
		lbc.Parsed.AcceptProxy = lbc.Ingress.KeepSourceIP()
	}
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
