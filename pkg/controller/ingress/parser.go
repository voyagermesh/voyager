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
	"github.com/appscode/voyager/pkg/controller/ingress/template"
	"github.com/flosch/pongo2"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func (lbc *EngressController) APISchema() string {
	if v, ok := lbc.Resource.Annotations[api.APISchema]; ok {
		return v
	}
	return api.APISchemaEngress
}

func (lbc *EngressController) SupportsLoadBalancerType() bool {
	return lbc.ProviderName == "aws" ||
		lbc.ProviderName == "gce" ||
		lbc.ProviderName == "gke" ||
		lbc.ProviderName == "azure" ||
		lbc.ProviderName == "minikube"
}

func (lbc *EngressController) parse() error {
	log.Infoln("Parsing new engress")
	if lbc.Resource == nil {
		log.Warningln("Config is nil, nothing to parse")
		return errors.New("no config found").Err()
	}
	lbc.parseOptions()
	lbc.parseSpec()
	return nil
}

func (lbc *EngressController) serviceEndpoints(name string, port intstr.IntOrString, hostNames []string) ([]*Endpoint, error) {
	log.Infoln("getting endpoints for ", lbc.Resource.Namespace, name, "port", port)

	// the following lines giving support to
	// serviceName.namespaceName or serviceName in the same namespace to the
	// ingress
	var namespace string = lbc.Resource.Namespace
	if strings.Contains(name, ".") {
		namespace = name[strings.Index(name, ".")+1:]
		name = name[:strings.Index(name, ".")]
	}

	log.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := lbc.KubeClient.Core().Services(namespace).Get(name)
	if err != nil {
		log.Warningln(err)
		return nil, err
	}

	if service.Spec.Type == kapi.ServiceTypeExternalName {
		// https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
		ep := Endpoint{
			Name:         "external",
			Port:         port.String(),
			ExternalName: service.Spec.ExternalName,
		}

		var resolver *api.DNSResolver
		var err error
		ep.UseDNSResolver, resolver, err = api.NewDNSResolver(*service)
		if err != nil {
			return nil, err
		}
		if ep.UseDNSResolver && resolver != nil {
			lbc.Parsed.DNSResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
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

func (lbc *EngressController) getEndpoints(s *kapi.Service, servicePort *kapi.ServicePort, hostNames []string) (eps []*Endpoint, err error) {
	ep, err := lbc.EndpointStore.GetServiceEndpoints(s)
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
						pod, err := lbc.Storage.PodStore.Pods(epAddress.TargetRef.Namespace).Get(epAddress.TargetRef.Name)
						if err != nil {
							log.Errorln("Error getting endpoind pod", err)
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

func (lbc *EngressController) generateTemplate() error {
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
	d, err := json.Marshal(s)
	if err != nil {
		return ctx, err
	}
	err = json.Unmarshal(d, &ctx)
	if err != nil {
		return ctx, errors.FromErr(err).Err()
	}
	return ctx, nil
}

func getSpecifiedPort(ports []kapi.ServicePort, port intstr.IntOrString) (*kapi.ServicePort, bool) {
	for _, p := range ports {
		if int(p.Port) == port.IntValue() {
			return &p, true
		}
	}
	return nil, false
}

// getTargetPort returns the numeric value of TargetPort
func getTargetPort(servicePort *kapi.ServicePort) int {
	return servicePort.TargetPort.IntValue()
}

func (lbc *EngressController) parseSpec() {
	log.Infoln("Parsing Engress specs")
	lbc.Ports = make([]int, 0)
	lbc.Parsed.DNSResolvers = make(map[string]*api.DNSResolver)

	if lbc.Resource.Spec.Backend != nil {
		log.Debugln("generating default backend", lbc.Resource.Spec.Backend.RewriteRule, lbc.Resource.Spec.Backend.HeaderRule)
		eps, _ := lbc.serviceEndpoints(lbc.Resource.Spec.Backend.ServiceName, lbc.Resource.Spec.Backend.ServicePort, lbc.Resource.Spec.Backend.HostNames)
		lbc.Parsed.DefaultBackend = &Backend{
			Name:      "default-backend",
			Endpoints: eps,

			BackendRules: lbc.Resource.Spec.Backend.BackendRule,
			RewriteRules: lbc.Resource.Spec.Backend.RewriteRule,
			HeaderRules:  lbc.Resource.Spec.Backend.HeaderRule,
		}
	}
	if len(lbc.Resource.Spec.TLS) > 0 {
		lbc.SecretNames = make([]string, 0)
		lbc.HostFilter = make([]string, 0)
		for _, secret := range lbc.Resource.Spec.TLS {
			lbc.SecretNames = append(lbc.SecretNames, secret.SecretName)
			lbc.HostFilter = append(lbc.HostFilter, secret.Hosts...)
		}
	}

	lbc.Parsed.HttpService = make([]*Service, 0)
	lbc.Parsed.HttpsService = make([]*Service, 0)
	lbc.Parsed.TCPService = make([]*TCPService, 0)

	var httpCount, httpsCount int
	for _, rule := range lbc.Resource.Spec.Rules {
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
			lbc.Ports = append(lbc.Ports, tcpSvc.Port.IntValue())
			log.Infoln(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort)
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

	if httpCount > 0 || (lbc.Resource.Spec.Backend != nil && httpsCount == 0) {
		lbc.Ports = append(lbc.Ports, 80)
	}

	if httpsCount > 0 {
		lbc.Ports = append(lbc.Ports, 443)
	}
}

func (lbc *EngressController) parseOptions() {
	if lbc.Resource == nil {
		log.Infoln("Config is nil, nothing to parse")
		return
	}
	log.Infoln("Parsing annotations.")
	lbc.Parsed.Sticky = lbc.Resource.StickySession()
	if len(lbc.Resource.Spec.TLS) > 0 {
		lbc.Parsed.SSLCert = true
	}

	lbc.Parsed.Stats = lbc.Resource.Stats()
	if lbc.Parsed.Stats {
		lbc.Parsed.StatsPort = lbc.Resource.StatsPort()
		if name := lbc.Resource.StatsSecretName(); len(name) > 0 {
			secret, err := lbc.KubeClient.Core().Secrets(lbc.Resource.ObjectMeta.Namespace).Get(name)
			if err == nil {
				lbc.Parsed.StatsUserName = string(secret.Data["username"])
				lbc.Parsed.StatsPassWord = string(secret.Data["password"])
			} else {
				lbc.Parsed.Stats = false
				log.Errorln("Error encountered while loading Stats secret", err)
			}
		}
	}

	if lbc.ProviderName == "aws" && lbc.Resource.LBType() == api.LBTypeLoadBalancer {
		lbc.Parsed.AcceptProxy = lbc.Resource.KeepSourceIP()
	}
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
