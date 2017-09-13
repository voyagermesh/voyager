package ingress

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/haproxy"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (c *controller) serviceEndpoints(dnsResolvers map[string]*api.DNSResolver, name string, port intstr.IntOrString, hostNames []string) (*haproxy.Backend, error) {
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
			dnsResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
			ep.CheckHealth = resolver.CheckHealth
		}
		return &haproxy.Backend{Endpoints: []*haproxy.Endpoint{&ep}}, nil
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		return nil, fmt.Errorf("service port %s unavailable for service %s", port.String(), service.Name)
	}
	return c.getEndpoints(service, p, hostNames)
}

func (c *controller) getEndpoints(s *apiv1.Service, servicePort *apiv1.ServicePort, hostNames []string) (*haproxy.Backend, error) {
	ep, err := c.EndpointsLister.Endpoints(s.Namespace).Get(s.Name)
	if err != nil {
		return nil, err
	}

	podList, err := c.KubeClient.CoreV1().Pods(s.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(s.Spec.Selector).String(),
	})
	if err != nil {
		return nil, err
	}
	pods := map[string]apiv1.Pod{}
	for _, pod := range podList.Items {
		pods[pod.Name] = pod
	}

	eps := make([]*haproxy.Endpoint, 0)
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
						pod, ok := pods[epAddress.TargetRef.Name]
						if !ok {
							log.Errorln("Error getting endpoint pod", err)
						} else {
							if pod.Annotations != nil {
								if val, ok := pod.Annotations[api.BackendWeight]; ok {
									ep.Weight, _ = strconv.Atoi(val)
								}
							}
						}
					}

					if s.Annotations != nil {
						ep.TLSOption = s.Annotations[api.BackendTLSOptions]
					}

					eps = append(eps, ep)
				}
			}
		}
	}
	return &haproxy.Backend{
		Endpoints:        eps,
		Sticky:           c.Ingress.Sticky() || isServiceSticky(s.Annotations),
		StickyCookieName: c.Ingress.StickySessionCookieName(),
	}, nil
}

func isServiceSticky(annotations map[string]string) bool {
	var sticky bool
	if annotations != nil {
		sticky, _ = strconv.ParseBool(annotations[api.StickySession])
	}
	return sticky
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

func getBackendName(r *api.Ingress, be api.IngressBackend) string {
	var seed string
	parts := strings.Split(be.ServiceName, ".")
	if len(parts) == 1 {
		seed = fmt.Sprintf("%s.%s:%d", parts[0], r.Namespace, be.ServicePort.IntValue())
	} else {
		seed = fmt.Sprintf("%s.%s:%d", parts[0], parts[1], be.ServicePort.IntValue()) // drop DNS labels following svcName, i.e.,  parts[2:]
	}
	return rand.WithUniqSuffix(seed)
}

func (c *controller) generateConfig() error {
	var td haproxy.TemplateData

	var nodePortSvc *apiv1.Service
	if c.Ingress.LBType() == api.LBTypeNodePort {
		var err error
		nodePortSvc, err = c.KubeClient.CoreV1().Services(c.Ingress.GetNamespace()).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
	}

	si := &haproxy.SharedInfo{}
	if c.Opt.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		si.AcceptProxy = c.Ingress.KeepSourceIP()
	}
	if c.Ingress.AcceptProxy() {
		si.AcceptProxy = true
	}

	dnsResolvers := make(map[string]*api.DNSResolver)
	if c.Ingress.Spec.Backend != nil {
		bk, err := c.serviceEndpoints(dnsResolvers, c.Ingress.Spec.Backend.ServiceName, c.Ingress.Spec.Backend.ServicePort, c.Ingress.Spec.Backend.HostNames)
		if err != nil {
			return err
		}
		si.DefaultBackend = &haproxy.Backend{
			Name:             "default-backend", // TODO: Use constant
			Endpoints:        bk.Endpoints,
			BackendRules:     c.Ingress.Spec.Backend.BackendRule,
			RewriteRules:     c.Ingress.Spec.Backend.RewriteRule,
			HeaderRules:      c.Ingress.Spec.Backend.HeaderRule,
			Sticky:           bk.Sticky,
			StickyCookieName: bk.StickyCookieName,
		}
	}

	if c.Ingress.AuthEnabled() {
		si.Auth = &haproxy.AuthConfig{
			Realm: c.Ingress.AuthRealm(),
		}

		secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(c.Ingress.AuthSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if secret.Data == nil {
			return fmt.Errorf("secret data missing")
		}
		si.Auth.Users, err = getAuthUsers(secret.Data)
		if err != nil {
			return err
		}
	}

	td.SharedInfo = si
	td.TimeoutDefaults = c.Ingress.Timeouts()
	td.OptionsDefaults = c.Ingress.HAProxyOptions()

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
		td.Stats = stats
	}

	td.HTTPService = make([]*haproxy.HTTPService, 0)
	td.TCPService = make([]*haproxy.TCPService, 0)

	type httpKey struct {
		Port     int
		NodePort int32
		UsesSSL  bool
	}
	httpServices := make(map[httpKey][]*haproxy.HTTPPath)
	for _, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			httpPaths := make([]*haproxy.HTTPPath, 0)
			for _, path := range rule.HTTP.Paths {
				bk, err := c.serviceEndpoints(dnsResolvers, path.Backend.ServiceName, path.Backend.ServicePort, path.Backend.HostNames)
				if err != nil {
					return err
				}
				if len(bk.Endpoints) > 0 {
					httpPaths = append(httpPaths, &haproxy.HTTPPath{
						Host: rule.Host,
						Path: path.Path,
						Backend: haproxy.Backend{
							Name:             getBackendName(c.Ingress, path.Backend.IngressBackend),
							Endpoints:        bk.Endpoints,
							BackendRules:     path.Backend.BackendRule,
							RewriteRules:     path.Backend.RewriteRule,
							HeaderRules:      path.Backend.HeaderRule,
							Sticky:           bk.Sticky,
							StickyCookieName: bk.StickyCookieName,
						},
					})
				}
			}

			var key httpKey
			if _, foundTLS := c.Ingress.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoTLS {
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

			if c.Ingress.LBType() == api.LBTypeNodePort && nodePortSvc != nil {
				for _, port := range nodePortSvc.Spec.Ports {
					if port.Port == int32(key.Port) {
						key.NodePort = port.NodePort
					}
				}
			}

			if v, ok := httpServices[key]; ok {
				httpServices[key] = append(v, httpPaths...)
			} else {
				httpServices[key] = httpPaths
			}
		} else if rule.TCP != nil {
			bk, err := c.serviceEndpoints(dnsResolvers, rule.TCP.Backend.ServiceName, rule.TCP.Backend.ServicePort, rule.TCP.Backend.HostNames)
			if err != nil {
				return err
			}
			if len(bk.Endpoints) > 0 {
				def := &haproxy.TCPService{
					SharedInfo:    si,
					FrontendName:  fmt.Sprintf("tcp-%s", rule.TCP.Port.String()),
					Host:          rule.Host,
					Port:          rule.TCP.Port.String(),
					ALPNOptions:   parseALPNOptions(rule.TCP.ALPN),
					FrontendRules: getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, rule.TCP.Port.IntValue()),
					Backend: haproxy.Backend{
						Name:             getBackendName(c.Ingress, rule.TCP.Backend),
						BackendRules:     rule.TCP.Backend.BackendRule,
						Endpoints:        bk.Endpoints,
						Sticky:           bk.Sticky,
						StickyCookieName: bk.StickyCookieName,
					},
				}
				if secretName, ok := c.Ingress.FindTLSSecret(rule.Host); ok && !rule.TCP.NoTLS {
					def.SecretName = secretName
				}
				td.TCPService = append(td.TCPService, def)
			}
		}
	}

	for key := range httpServices {
		value := httpServices[key]
		td.HTTPService = append(td.HTTPService, &haproxy.HTTPService{
			SharedInfo:    si,
			FrontendName:  fmt.Sprintf("http-%d", key.Port),
			Port:          key.Port,
			FrontendRules: getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, key.Port),
			NodePort:      key.NodePort,
			UsesSSL:       key.UsesSSL,
			Paths:         value,
		})
	}

	td.DNSResolvers = make([]*api.DNSResolver, 0)
	for k := range dnsResolvers {
		td.DNSResolvers = append(td.DNSResolvers, dnsResolvers[k])
	}

	if jb, err := json.MarshalIndent(&td, "", "  "); err != nil {
		log.Debugf("Rendering haproxy.cfg for Ingress %s@%s using data:", c.Ingress.Name, c.Ingress.Namespace, string(jb))
	}
	if cfg, err := haproxy.RenderConfig(td); err != nil {
		return err
	} else {
		c.HAProxyConfig = cfg
		log.Debugf("Generated haproxy.cfg for Ingress %s@%s:", c.Ingress.Name, c.Ingress.Namespace, cfg)
	}
	return nil
}

func getAuthUsers(data map[string][]byte) (map[string][]haproxy.AuthUser, error) {
	ret := make(map[string][]haproxy.AuthUser, 0)
	for name, data := range data {
		users := make([]haproxy.AuthUser, 0)
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) <= 0 {
				continue
			}
			sep := strings.Index(line, ":")
			if sep == -1 {
				return nil, fmt.Errorf("Missing ':' on userlist")
			}
			userName := line[0:sep]
			if userName == "" {
				return nil, fmt.Errorf("Missing username on userlist")
			}
			if sep == len(line)-1 || line[sep:] == "::" {
				return nil, fmt.Errorf("Missing '%v' password on userlist", userName)
			}
			user := haproxy.AuthUser{}
			// if usr::pwd
			if string(line[sep+1]) == ":" {
				pass, err := crypt.NewFromHash(sha512_crypt.MagicPrefix).Generate([]byte(line[sep+2:]), nil)
				if err != nil {
					return nil, err
				}
				user = haproxy.AuthUser{
					Username:  userName,
					Password:  pass,
					Encrypted: true,
				}
			} else {
				user = haproxy.AuthUser{
					Username:  userName,
					Password:  line[sep+1:],
					Encrypted: true,
				}
			}
			users = append(users, user)
		}
		ret[name] = users
	}
	return ret, nil
}

func getFrontendRulesForPort(rules []api.FrontendRule, port int) []string {
	for _, rule := range rules {
		if rule.Port.IntValue() == port {
			return rule.Rules
		}
	}
	return []string{}
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}
