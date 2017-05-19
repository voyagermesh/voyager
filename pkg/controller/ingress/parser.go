package ingress

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/go/arrays"
	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/ingress/template"
	"github.com/flosch/pongo2"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func (lbc *EngressController) parse() error {
	log.Infoln("Parsing new engress")
	if lbc.Config == nil {
		log.Warningln("Config is nil, nothing to parse")
		return errors.New("no config found").Err()
	}
	lbc.parseOptions()
	lbc.parseSpec()
	lbc.Options.ConfigMapName = VoyagerPrefix + lbc.Config.Name
	return nil
}

func (lbc *EngressController) serviceEndpoints(name string, port intstr.IntOrString, hostNames []string) ([]*Endpoint, error) {
	log.Infoln("getting endpoints for ", lbc.Config.Namespace, name, "port", port)

	// the following lines giving support to
	// serviceName.namespaceName or serviceName in the same namespace to the
	// ingress
	var namespace string = lbc.Config.Namespace
	if strings.Contains(name, ".") {
		namespace = name[strings.Index(name, ".")+1:]
		name = name[:strings.Index(name, ".")]
	}

	log.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := lbc.KubeClient.Core().Services(namespace).Get(name)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		return nil, errors.New("service port unavaiable").Err()
	}
	return lbc.getEndpoints(service, p, hostNames)
}

func (lbc *EngressController) getEndpoints(s *kapi.Service, servicePort *kapi.ServicePort, hostNames []string) (eps []*Endpoint, err error) {
	ep, err := lbc.EndpointStore.GetServiceEndpoints(s)
	if err != nil {
		return nil, errors.FromErr(err).Err()
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
								if val, ok := pod.Annotations[LoadBalancerBackendWeight]; ok {
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

	lbc.Options.ConfigData = stringutil.Fmt(r)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.Infoln("Template genareted for HAProxy")
	log.Infoln(lbc.Options.ConfigData)
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
	lbc.Options.Ports = make([]int, 0)
	if lbc.Config.Spec.Backend != nil {
		log.Debugln("generating default backend", lbc.Config.Spec.Backend.RewriteRule, lbc.Config.Spec.Backend.HeaderRule)
		eps, _ := lbc.serviceEndpoints(lbc.Config.Spec.Backend.ServiceName, lbc.Config.Spec.Backend.ServicePort, lbc.Config.Spec.Backend.HostNames)
		lbc.Parsed.DefaultBackend = &Backend{
			Name:      "default-backend",
			Endpoints: eps,

			BackendRules: lbc.Config.Spec.Backend.BackendRule,
			RewriteRules: lbc.Config.Spec.Backend.RewriteRule,
			HeaderRules:  lbc.Config.Spec.Backend.HeaderRule,
		}
	}
	if len(lbc.Config.Spec.TLS) > 0 {
		lbc.Options.SecretNames = make([]string, 0)
		lbc.HostFilter = make([]string, 0)
		for _, secret := range lbc.Config.Spec.TLS {
			lbc.Options.SecretNames = append(lbc.Options.SecretNames, secret.SecretName)
			lbc.HostFilter = append(lbc.HostFilter, secret.Hosts...)
		}
	}

	lbc.Parsed.HttpService = make([]*Service, 0)
	lbc.Parsed.HttpsService = make([]*Service, 0)
	lbc.Parsed.TCPService = make([]*TCPService, 0)

	var httpCount, httpsCount int
	for _, rule := range lbc.Config.Spec.Rules {
		host := rule.Host
		if rule.HTTP != nil {
			if ok, _ := arrays.Contains(lbc.HostFilter, host); ok {
				httpsCount++
			} else {
				httpCount++
			}

			for _, svc := range rule.HTTP.Paths {
				def := &Service{
					Name:     "service-" + rand.Characters(6),
					Host:     host,
					AclMatch: svc.Path,
				}

				eps, err := lbc.serviceEndpoints(svc.Backend.ServiceName, svc.Backend.ServicePort, svc.Backend.HostNames)
				def.Backends = &Backend{
					Name:         "backend-" + rand.Characters(5),
					Endpoints:    eps,
					BackendRules: svc.Backend.BackendRule,
					RewriteRules: svc.Backend.RewriteRule,
					HeaderRules:  svc.Backend.HeaderRule,
				}

				log.Debugln("Got endpoints", len(eps))
				if len(eps) > 0 && err == nil {
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
			lbc.Options.Ports = append(lbc.Options.Ports, tcpSvc.Port.IntValue())
			def := &TCPService{
				Name:        "service-" + rand.Characters(6),
				Host:        host,
				Port:        tcpSvc.Port.String(),
				SecretName:  tcpSvc.SecretName,
				ALPNOptions: parseALPNOptions(tcpSvc.ALPN),
			}
			log.Infoln(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort)
			eps, err := lbc.serviceEndpoints(tcpSvc.Backend.ServiceName, tcpSvc.Backend.ServicePort, tcpSvc.Backend.HostNames)
			def.Backends = &Backend{
				Name:         "backend-" + rand.Characters(5),
				BackendRules: tcpSvc.Backend.BackendRule,
				Endpoints:    eps,
			}

			log.Debugln("Got endpoints", len(eps))
			if len(eps) > 0 && err == nil {
				lbc.Parsed.TCPService = append(lbc.Parsed.TCPService, def)
				lbc.Options.SecretNames = append(lbc.Options.SecretNames, def.SecretName)
			}
		}
	}

	if httpCount > 0 || (lbc.Config.Spec.Backend != nil && httpsCount == 0) {
		lbc.Options.Ports = append(lbc.Options.Ports, 80)
	}

	if httpsCount > 0 {
		lbc.Options.Ports = append(lbc.Options.Ports, 443)
	}

	//parse stat
	if lbc.Parsed.Stats {
		lbc.Options.Ports = append(lbc.Options.Ports, StatPort)
	}
}

func (lbc *EngressController) parseOptions() {
	if lbc.Config == nil {
		log.Infoln("Config is nil, nothing to parse")
		return
	}
	log.Infoln("Parsing annotations.")
	lbc.Options.annotations = annotation(lbc.Config.ObjectMeta.Annotations)
	lbc.Parsed.Sticky = lbc.Options.annotations.StickySession()
	if len(lbc.Config.Spec.TLS) > 0 {
		lbc.Parsed.SSLCert = true
	}

	lbc.Parsed.Stats = lbc.Options.annotations.Stats()
	if lbc.Parsed.Stats {
		secret, err := lbc.KubeClient.Core().Secrets(lbc.Config.ObjectMeta.Namespace).Get(lbc.Options.annotations.StatsSecretName())
		if err == nil {
			lbc.Parsed.StatsUserName = string(secret.Data["username"])
			lbc.Parsed.StatsPassWord = string(secret.Data["password"])
		} else {
			log.Errorln("Error encountered while loading secret,", err)
		}
	}

	lbc.Options.LBType = lbc.Options.annotations.LBType()
	lbc.Options.Replicas = lbc.Options.annotations.Replicas()
	lbc.Options.DaemonNodeSelector = ParseNodeSelector(lbc.Options.annotations.DaemonNodeSelector())
	lbc.Options.LoadBalancerIP = lbc.Options.annotations.LoadBalancerIP()
	lbc.Options.LoadBalancerPersist = lbc.Options.annotations.LoadBalancerPersist()
	log.Infoln("Got LBType", lbc.Options.LBType)
}

// ref: https://github.com/kubernetes/kubernetes/blob/078238a461a0872a8eacb887fbb3d0085714604c/staging/src/k8s.io/apiserver/pkg/apis/example/v1/types.go#L134
func ParseNodeSelector(labels string) map[string]string {
	selectorMap := make(map[string]string)
	for _, label := range strings.Split(labels, ",") {
		label = strings.TrimSpace(label)
		if len(label) > 0 && strings.Contains(label, "=") {
			data := strings.SplitN(label, "=", 2)
			if len(data) >= 2 {
				if len(data[0]) > 0 && len(data[1]) > 0 {
					selectorMap[data[0]] = data[1]
				}
			}
		}
	}
	return selectorMap
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
