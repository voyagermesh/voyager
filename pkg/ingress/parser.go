package ingress

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/haproxy"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (c *controller) serviceEndpoints(dnsResolvers map[string]*api.DNSResolver, userLists map[string]haproxy.UserList, name string, port intstr.IntOrString, hostNames []string) (*haproxy.Backend, error) {
	c.logger.Infoln("getting endpoints for ", c.Ingress.Namespace, name, "port", port)

	// the following lines giving support to
	// serviceName.namespaceName or serviceName in the same namespace to the
	// ingress
	var namespace string = c.Ingress.Namespace
	if strings.Contains(name, ".") {
		namespace = name[strings.Index(name, ".")+1:]
		name = name[:strings.Index(name, ".")]
	}

	c.logger.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := c.ServiceLister.Services(namespace).Get(name)
	if err != nil {
		c.logger.Warningln(err)
		return nil, err
	}

	if service.Spec.Type == apiv1.ServiceTypeExternalName {
		c.logger.Infof("Found ServiceType ExternalName for service %s, Checking DNS resolver options", service.Name)
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
	return c.getEndpoints(service, p, hostNames, userLists)
}

func (c *controller) getEndpoints(svc *apiv1.Service, servicePort *apiv1.ServicePort, hostNames []string, userLists map[string]haproxy.UserList) (*haproxy.Backend, error) {
	ep, err := c.EndpointsLister.Endpoints(svc.Namespace).Get(svc.Name)
	if err != nil {
		return nil, err
	}

	podList, err := c.KubeClient.CoreV1().Pods(svc.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
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
					c.logger.Debugf("Target port %s empty for service %s. skipping.", servicePort.String(), svc.Name)
					continue
				}
			}

			c.logger.Infof("Found target port %s for service %s", targetPort, svc.Name)
			for _, epAddress := range ss.Addresses {
				if isForwardable(hostNames, epAddress.Hostname) {
					ep := &haproxy.Endpoint{
						Name: getEndpointName(epAddress),
						IP:   epAddress.IP,
						Port: targetPort,
					}
					if epAddress.TargetRef != nil {
						// Use PodList via service selector
						pod, ok := pods[epAddress.TargetRef.Name]
						if !ok {
							c.logger.Errorln("Error getting endpoint pod", err)
						} else {
							if pod.Annotations != nil {
								if val, ok := pod.Annotations[api.BackendWeight]; ok {
									ep.Weight, _ = strconv.Atoi(val)
								}
								if val, ok := pod.Annotations[api.MaxConnections]; ok {
									ep.MaxConnections, _ = strconv.Atoi(val)
								}
							}
						}
					}

					if svc.Annotations != nil {
						ep.TLSOption = svc.Annotations[api.BackendTLSOptions]
					}

					eps = append(eps, ep)
				}
			}
		}
	}
	return &haproxy.Backend{
		BasicAuth:        c.getServiceAuth(userLists, svc),
		Endpoints:        eps,
		Sticky:           c.Ingress.Sticky() || isServiceSticky(svc.Annotations),
		StickyCookieName: c.Ingress.StickySessionCookieName(),
		StickyCookieHash: c.Ingress.StickySessionCookieHashType(),
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

	si := &haproxy.SharedInfo{
		EnableCORS:            c.Ingress.EnableCORS(),
		ProxyBodySize:         c.Ingress.ProxyBodySize(),
		EnableHSTS:            c.Ingress.EnableHSTS(),
		HSTSMaxAge:            c.Ingress.HSTSMaxAge(),
		HSTSPreload:           c.Ingress.HSTSPreload(),
		HSTSIncludeSubDomains: c.Ingress.HSTSIncludeSubDomains(),
		WhitelistSourceRange:  c.Ingress.WhitelistSourceRange(),
		MaxConnections:        c.Ingress.MaxConnections(),
		ForceMatchServicePort: c.Ingress.ForceServicePort(),
		Limit: &haproxy.Limit{
			Connection: c.Ingress.LimitConnections(),
		},
		SSLRedirect: c.Ingress.ForceSSLRedirect() || c.Ingress.SSLRedirect(),
	}

	if val := c.Ingress.LimitRPM(); val > 0 {
		si.Limit.TimeSecond = 60
		si.Limit.Rate = val
	} else if val := c.Ingress.LimitRPS(); val > 0 {
		si.Limit.TimeSecond = 1
		si.Limit.Rate = val
	}

	if c.Opt.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		si.AcceptProxy = c.Ingress.KeepSourceIP()
	}
	if c.Ingress.AcceptProxy() {
		si.AcceptProxy = true
	}

	userLists := make(map[string]haproxy.UserList)
	var globalBasic *haproxy.BasicAuth
	if c.Ingress.BasicAuthEnabled() {
		globalBasic = &haproxy.BasicAuth{
			Realm: c.Ingress.AuthRealm(),
		}
		secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(c.Ingress.AuthSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		globalBasic.UserLists, err = getBasicAuthUsers(userLists, secret)
		if err != nil {
			return err
		}
	}
	var globalTLS *haproxy.TLSAuth
	if c.Ingress.AuthTLSSecret() != "" {
		var err error
		globalTLS, err = c.getTLSAuth(&api.TLSAuth{
			SecretName:   c.Ingress.AuthTLSSecret(),
			VerifyClient: c.Ingress.AuthTLSVerifyClient(),
			ErrorPage:    c.Ingress.AuthTLSErrorPage(),
		})
		if err != nil {
			return err
		}
	}

	dnsResolvers := make(map[string]*api.DNSResolver)
	if c.Ingress.Spec.Backend != nil {
		bk, err := c.serviceEndpoints(dnsResolvers, userLists, c.Ingress.Spec.Backend.ServiceName, c.Ingress.Spec.Backend.ServicePort, c.Ingress.Spec.Backend.HostNames)
		if err != nil {
			return err
		}
		si.DefaultBackend = &haproxy.Backend{
			Name:             "default-backend", // TODO: Use constant
			BasicAuth:        bk.BasicAuth,
			Endpoints:        bk.Endpoints,
			BackendRules:     c.Ingress.Spec.Backend.BackendRule,
			RewriteRules:     c.Ingress.Spec.Backend.RewriteRule,
			HeaderRules:      c.Ingress.Spec.Backend.HeaderRule,
			Sticky:           bk.Sticky,
			StickyCookieName: bk.StickyCookieName,
			StickyCookieHash: bk.StickyCookieHash,
		}
		if globalBasic != nil {
			si.DefaultBackend.BasicAuth = globalBasic
		}
	}

	if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
		errorFiles, err := c.getErrorFiles()
		if err != nil {
			return err
		}
		td.ErrorFiles = errorFiles
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
				return fmt.Errorf("failed to load stats secret for ingress %s@%s", c.Ingress.Name, c.Ingress.Namespace)
			}
		}
		td.Stats = stats
	}

	td.HTTPService = make([]*haproxy.HTTPService, 0)
	td.TCPService = make([]*haproxy.TCPService, 0)

	type httpKey struct {
		Port       int
		NodePort   int32
		OffloadSSL bool
	}
	httpServices := make(map[httpKey][]*haproxy.HTTPPath)
	for _, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			httpPaths := make([]*haproxy.HTTPPath, 0)
			for _, path := range rule.HTTP.Paths {
				bk, err := c.serviceEndpoints(dnsResolvers, userLists, path.Backend.ServiceName, path.Backend.ServicePort, path.Backend.HostNames)
				if err != nil {
					return err
				}
				if len(bk.Endpoints) > 0 {
					httpPaths = append(httpPaths, &haproxy.HTTPPath{
						Host: rule.Host,
						Path: path.Path,
						Backend: haproxy.Backend{
							Name:             getBackendName(c.Ingress, path.Backend.IngressBackend),
							BasicAuth:        bk.BasicAuth,
							Endpoints:        bk.Endpoints,
							BackendRules:     path.Backend.BackendRule,
							RewriteRules:     path.Backend.RewriteRule,
							HeaderRules:      path.Backend.HeaderRule,
							Sticky:           bk.Sticky,
							StickyCookieName: bk.StickyCookieName,
							StickyCookieHash: bk.StickyCookieHash,
						},
					})
				}
			}

			var key httpKey
			if _, foundTLS := c.Ingress.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoTLS && !c.Ingress.SSLPassthrough() {
				key.OffloadSSL = true
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					key.Port = port
				} else {
					key.Port = 443
				}
			} else if foundTLS && c.Ingress.SSLPassthrough() && !rule.HTTP.NoTLS {
				// If SSL Passthrough is enabled keep 443 open just don't offload ssl
				key.OffloadSSL = false
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					key.Port = port
				} else {
					key.Port = 443
				}
			} else {
				key.OffloadSSL = false
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
			bk, err := c.serviceEndpoints(dnsResolvers, userLists, rule.TCP.Backend.ServiceName, rule.TCP.Backend.ServicePort, rule.TCP.Backend.HostNames)
			if err != nil {
				return err
			}
			if len(bk.Endpoints) > 0 {
				fr := getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, rule.TCP.Port.IntValue())
				def := &haproxy.TCPService{
					SharedInfo:    si,
					FrontendName:  fmt.Sprintf("tcp-%s", rule.TCP.Port.String()),
					Host:          rule.Host,
					Port:          rule.TCP.Port.String(),
					ALPNOptions:   parseALPNOptions(rule.TCP.ALPN),
					FrontendRules: fr.Rules,
					Backend: haproxy.Backend{
						Name:             getBackendName(c.Ingress, rule.TCP.Backend),
						BackendRules:     rule.TCP.Backend.BackendRule,
						Endpoints:        bk.Endpoints,
						Sticky:           bk.Sticky,
						StickyCookieName: bk.StickyCookieName,
						StickyCookieHash: bk.StickyCookieHash,
					},
				}
				if globalTLS != nil {
					def.TLSAuth = globalTLS
				} else if fr.Auth != nil && fr.Auth.TLS != nil {
					htls, err := c.getTLSAuth(fr.Auth.TLS)
					if err != nil {
						return err
					}

					if htls != nil {
						def.TLSAuth = htls
					}
				}

				if ref, ok := c.Ingress.FindTLSSecret(rule.Host); ok && !rule.TCP.NoTLS {
					if ref.Kind == api.ResourceKindCertificate {
						crd, err := c.VoyagerClient.Certificates(c.Ingress.Namespace).Get(ref.Name, metav1.GetOptions{})
						if err == nil {
							def.CertFile = crd.SecretName() + ".pem"
						}
					} else {
						def.CertFile = ref.Name + ".pem" // Add file extension too
					}
				}
				td.TCPService = append(td.TCPService, def)
			}
		}
	}

	tp80 := false
	for key := range httpServices {
		value := httpServices[key]
		if key.Port == 80 {
			tp80 = true
		}
		fr := getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, key.Port)
		srv := &haproxy.HTTPService{
			SharedInfo:    si,
			FrontendName:  fmt.Sprintf("http-%d", key.Port),
			Port:          key.Port,
			FrontendRules: fr.Rules,
			NodePort:      key.NodePort,
			OffloadSSL:    key.OffloadSSL,
			Paths:         value,
		}

		if globalBasic != nil {
			srv.BasicAuth = globalBasic
			srv.RemoveBackendAuth()
		} else if globalTLS != nil {
			srv.TLSAuth = globalTLS
			srv.RemoveBackendAuth()
		} else if fr.Auth != nil && fr.Auth.Basic != nil {
			srv.BasicAuth = &haproxy.BasicAuth{
				Realm: fr.Auth.Basic.Realm,
			}
			secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(fr.Auth.Basic.SecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			srv.BasicAuth.UserLists, err = getBasicAuthUsers(userLists, secret)
			if err != nil {
				return err
			}
			srv.RemoveBackendAuth()
		} else if fr.Auth != nil && fr.Auth.TLS != nil {
			htls, err := c.getTLSAuth(fr.Auth.TLS)
			if err != nil {
				return err
			}
			if htls != nil {
				srv.TLSAuth = htls
			}
			srv.RemoveBackendAuth()
		}
		td.HTTPService = append(td.HTTPService, srv)
	}
	if !tp80 && c.Ingress.SSLRedirect() {
		td.HTTPService = append(td.HTTPService, &haproxy.HTTPService{
			SharedInfo:   si,
			FrontendName: fmt.Sprintf("http-%d", 80),
			Port:         80,
		})
	}

	td.DNSResolvers = make([]*api.DNSResolver, 0, len(dnsResolvers))
	for k := range dnsResolvers {
		td.DNSResolvers = append(td.DNSResolvers, dnsResolvers[k])
	}

	td.UserLists = make([]haproxy.UserList, 0, len(userLists))
	for k := range userLists {
		td.UserLists = append(td.UserLists, userLists[k])
	}

	if jb, err := json.MarshalIndent(&td, "", "  "); err != nil {
		c.logger.Debugf("Rendering haproxy.cfg for Ingress %s@%s using data:", c.Ingress.Name, c.Ingress.Namespace, string(jb))
	}
	if cfg, err := haproxy.RenderConfig(td); err != nil {
		return err
	} else {
		c.HAProxyConfig = cfg
		c.logger.Debugf("Generated haproxy.cfg for Ingress %s/%s", c.Ingress.Namespace, c.Ingress.Name)
	}
	return nil
}

func getBasicAuthUsers(userLists map[string]haproxy.UserList, sec *apiv1.Secret) ([]string, error) {
	listNames := make([]string, 0)

	for name, data := range sec.Data {
		listName := sec.Name + "-" + name
		listNames = append(listNames, listName)

		if _, found := userLists[listName]; found {
			continue
		}

		users := make([]haproxy.AuthUser, 0)
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) <= 0 {
				continue
			}
			sep := strings.Index(line, ":")
			if sep == -1 {
				return nil, fmt.Errorf("missing ':' on userlist")
			}
			userName := line[0:sep]
			if userName == "" {
				return nil, fmt.Errorf("missing username on userlist")
			}
			if sep == len(line)-1 || line[sep:] == "::" {
				return nil, fmt.Errorf("missing '%v' password on userlist", userName)
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
		userLists[listName] = haproxy.UserList{Name: listName, Users: users}
	}
	return listNames, nil
}

func getFrontendRulesForPort(rules []api.FrontendRule, port int) api.FrontendRule {
	for _, rule := range rules {
		if rule.Port.IntValue() == port {
			return rule
		}
	}
	return api.FrontendRule{}
}

func parseALPNOptions(opt []string) string {
	if len(opt) <= 0 {
		return ""
	}
	return "alpn " + strings.Join(opt, ",")
}

func getEndpointName(ep apiv1.EndpointAddress) string {
	if ep.TargetRef != nil {
		return "pod-" + ep.TargetRef.Name
	}
	return "pod-" + ep.IP
}

func (c *controller) getServiceAuth(userLists map[string]haproxy.UserList, svc *apiv1.Service) *haproxy.BasicAuth {
	// Check auth type is basic; other auth mode is not supported
	authType, ok := svc.Annotations[api.AuthType]
	if !ok || authType != "basic" {
		return nil
	}

	authSecret, ok := svc.Annotations[api.AuthSecret]
	if !ok {
		return nil
	}
	secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(authSecret, metav1.GetOptions{})
	if err != nil || secret.Data == nil {
		return nil
	}

	userList, err := getBasicAuthUsers(userLists, secret)
	if err != nil {
		return nil
	}
	return &haproxy.BasicAuth{
		Realm:     svc.Annotations[api.AuthRealm],
		UserLists: userList,
	}
}

func (c *controller) getErrorFiles() ([]*haproxy.ErrorFile, error) {
	configMap, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.ErrorFilesConfigMapName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	commands := sets.NewString("errorfile", "errorloc", "errorloc302", "errorloc303")
	codes := []string{"200", "400", "403", "405", "408", "429", "500", "502", "503", "504"}
	errorFiles := make([]*haproxy.ErrorFile, 0, len(codes))
	for _, statusCode := range codes {
		if _, found := configMap.Data[statusCode+".http"]; found {
			errorFiles = append(errorFiles, &haproxy.ErrorFile{
				StatusCode: statusCode,
				Command:    ErrorFilesCommand,
				Value:      fmt.Sprintf("%s/%s.http", ErrorFilesLocation, statusCode),
			})
		} else if val, found := configMap.Data[statusCode]; found {
			parts := strings.SplitN(val, " ", 2)
			if len(parts) < 2 {
				return nil, err
			}
			if !commands.Has(parts[0]) {
				return nil, fmt.Errorf("found unknown errofile command %s", parts[0])
			}
			errorFiles = append(errorFiles, &haproxy.ErrorFile{
				StatusCode: statusCode,
				Command:    parts[0],
				Value:      parts[1],
			})
		}
	}
	return errorFiles, nil
}

func (c *controller) getTLSAuth(cfg *api.TLSAuth) (*haproxy.TLSAuth, error) {
	tlsAuthSec, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(cfg.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if _, ok := tlsAuthSec.Data["ca.crt"]; !ok {
		return nil, fmt.Errorf("key ca.crt not found in TLSAuthSecret %s", tlsAuthSec.Name)
	}

	htls := &haproxy.TLSAuth{
		CAFile:       cfg.SecretName + "-ca.crt",
		VerifyClient: string(cfg.VerifyClient),
		Headers:      cfg.Headers,
		ErrorPage:    cfg.ErrorPage,
	}
	if _, ok := tlsAuthSec.Data["crl.pem"]; ok {
		htls.CRLFile = cfg.SecretName + "-crl.pem"
	}
	if u, err := url.Parse(cfg.ErrorPage); err == nil {
		htls.ErrorPath = u.Path
	}
	if htls.VerifyClient == "" {
		htls.VerifyClient = string(api.TLSAuthVerifyRequired)
	}

	return htls, nil
}
