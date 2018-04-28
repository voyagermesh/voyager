package ingress

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	hpi "github.com/appscode/voyager/pkg/haproxy/api"
	"github.com/appscode/voyager/pkg/haproxy/template"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	"github.com/pkg/errors"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (c *controller) serviceEndpoints(dnsResolvers map[string]*api.DNSResolver, userLists map[string]hpi.UserList, bkSvc string, port intstr.IntOrString, hostNames []string) (*hpi.Backend, error) {
	c.logger.Infoln("getting endpoints for ", c.Ingress.Namespace, bkSvc, "port", port)

	name := bkSvc
	namespace := c.Ingress.Namespace
	if strings.Contains(name, ".") {
		idx := strings.Index(name, ".")
		namespace = name[idx+1:]
		name = name[:idx]
	}
	if c.cfg.RestrictToOperatorNamespace && namespace != c.cfg.OperatorNamespace {
		return nil, errors.Errorf("can't use service %s as backend, since voyager operator is restricted namespace %s", bkSvc, c.cfg.OperatorNamespace)
	}

	c.logger.Infoln("looking for services in namespace", namespace, "with name", name)
	service, err := c.ServiceLister.Services(namespace).Get(name)
	if kerr.IsNotFound(err) {
		return nil, kerr.NewNotFound(core.Resource("service"), namespace+"/"+name)
	} else if err != nil {
		return nil, err
	}

	if service.Spec.Type == core.ServiceTypeExternalName {
		c.logger.Infof("Found ServiceType ExternalName for service %s, Checking DNS resolver options", service.Name)
		// https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
		ep := hpi.Endpoint{
			Name:         "external",
			Port:         port.String(),
			ExternalName: service.Spec.ExternalName,
		}

		var resolver *api.DNSResolver
		var err error
		ep.UseDNSResolver, resolver, err = api.DNSResolverForService(*service)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if ep.UseDNSResolver && resolver != nil {
			dnsResolvers[resolver.Name] = resolver
			ep.DNSResolver = resolver.Name
			ep.CheckHealth = resolver.CheckHealth
		}
		return &hpi.Backend{Endpoints: []*hpi.Endpoint{&ep}}, nil
	}
	p, ok := getSpecifiedPort(service.Spec.Ports, port)
	if !ok {
		return nil, errors.Errorf("service port %s unavailable for service %s", port.String(), service.Name)
	}
	return c.getEndpoints(service, p, hostNames, userLists)
}

func (c *controller) getEndpoints(svc *core.Service, servicePort *core.ServicePort, hostNames []string, userLists map[string]hpi.UserList) (*hpi.Backend, error) {
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
	pods := map[string]core.Pod{}
	for _, pod := range podList.Items {
		pods[pod.Name] = pod
	}

	eps := make([]*hpi.Endpoint, 0)
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
					ep := &hpi.Endpoint{
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
						if svc.Annotations[api.CheckHealth] == "true" {
							ep.CheckHealth = true
							ep.CheckHealthPort = svc.Annotations[api.CheckHealthPort]
						}
						ep.SendProxy = api.ProxyProtocolCommand(svc.Annotations[api.SendProxy])
					}

					eps = append(eps, ep)
				}
			}
		}
	}
	return &hpi.Backend{
		BasicAuth:        c.getServiceAuth(userLists, svc),
		Endpoints:        eps,
		Sticky:           c.Ingress.Sticky() || isServiceSticky(svc.Annotations),
		StickyCookieName: c.Ingress.StickySessionCookieName(),
		StickyCookieHash: c.Ingress.StickySessionCookieHashType(),
	}, nil
}

func isServiceSticky(annotations map[string]string) bool {
	v, _ := meta.GetStringValue(annotations, api.IngressAffinity)
	return v == "cookie"
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

func getSpecifiedPort(ports []core.ServicePort, port intstr.IntOrString) (*core.ServicePort, bool) {
	for _, p := range ports {
		if int(p.Port) == port.IntValue() {
			return &p, true
		}
	}
	return nil, false
}

func getFrontendName(proto, addr string, port int) string {
	switch addr {
	case ``, `*`:
		return fmt.Sprintf("%s-0_0_0_0-%d", proto, port)
	default:
		return fmt.Sprintf("%s-%s-%d", proto, strings.Replace(addr, ".", "_", 3), port)
	}
}

func getBackendName(r *api.Ingress, be api.IngressBackend) string {
	parts := strings.Split(be.ServiceName, ".")
	if len(parts) == 1 {
		return fmt.Sprintf("%s.%s:%d", parts[0], r.Namespace, be.ServicePort.IntValue())
	} else {
		return fmt.Sprintf("%s.%s:%d", parts[0], parts[1], be.ServicePort.IntValue()) // drop DNS labels following svcName, i.e.,  parts[2:]
	}
}

// ref: https://github.com/jcmoraisjr/haproxy-ingress/pull/57
// ref: https://github.com/jcmoraisjr/haproxy-ingress/blob/939bd129c86d9b27b12e6d7a50c799d8496ab8f3/rootfs/etc/haproxy/template/haproxy.tmpl#L318
func (c *controller) rewriteTarget(path string, rewriteRules []string) []string {
	target := c.Ingress.RewriteTarget()
	switch target {
	case "":
		return rewriteRules
	case "/":
		// ^([^\ :]*)\ {{ $location.Path }}/?(.*$) \1\ {{ $rewriteTarget }}\2
		rule := fmt.Sprintf(`^([^\ :]*)\ %s/?(.*$) \1\ %s\2`, path, target)
		return append([]string{rule}, rewriteRules...)
	default:
		// ^([^\ :]*)\ {{ $location.Path }}(.*$) \1\ {{ $rewriteTarget }}{{ if hasSuffix $location.Path "/" }}/{{ end }}\2
		if strings.HasSuffix(path, "/") {
			target = target + "/"
		}
		rule := fmt.Sprintf(`^([^\ :]*)\ %s(.*$) \1\ %s\2`, path, target)
		return append([]string{rule}, rewriteRules...)
	}
}

type hostBinder struct {
	Address string
	Port    int
}
type httpInfo struct {
	OffloadSSL bool
	Hosts      map[string][]*hpi.HTTPPath
}
type tcpInfo struct {
	CertFile    string
	ALPNOptions string
	Hosts       []*hpi.TCPHost
}

func (c *controller) generateConfig() error {
	if c.Ingress.SSLPassthrough() {
		if err := c.convertRulesForSSLPassthrough(); err != nil {
			return err
		}
	}

	// assign address
	for _, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil && rule.HTTP.Address == "" {
			rule.HTTP.Address = `*`
		} else if rule.TCP != nil && rule.TCP.Address == "" {
			rule.TCP.Address = `*`
		}
	}

	var td hpi.TemplateData

	si := &hpi.SharedInfo{
		CORSConfig: hpi.CORSConfig{
			CORSEnabled:          c.Ingress.EnableCORS(),
			CORSAllowedOrigin:    c.Ingress.AllowedCORSOrigin(),
			CORSAllowedMethods:   c.Ingress.AllowedCORSMethods(),
			CORSAllowedHeaders:   c.Ingress.AllowedCORSHeaders(),
			CORSAllowCredentials: c.Ingress.AllowCORSCred(),
		},
		ProxyBodySize:         c.Ingress.ProxyBodySize(),
		EnableHSTS:            c.Ingress.EnableHSTS(),
		HSTSMaxAge:            c.Ingress.HSTSMaxAge(),
		HSTSPreload:           c.Ingress.HSTSPreload(),
		HSTSIncludeSubDomains: c.Ingress.HSTSIncludeSubDomains(),
		WhitelistSourceRange:  c.Ingress.WhitelistSourceRange(),
		MaxConnections:        c.Ingress.MaxConnections(),
		UseNodePort:           c.Ingress.UseNodePort(),
		Limit: &hpi.Limit{
			Connection: c.Ingress.LimitConnections(),
		},
	}

	if val := c.Ingress.LimitRPM(); val > 0 {
		si.Limit.TimeSecond = 60
		si.Limit.Rate = val
	} else if val := c.Ingress.LimitRPS(); val > 0 {
		si.Limit.TimeSecond = 1
		si.Limit.Rate = val
	}

	if c.cfg.CloudProvider == "aws" && c.Ingress.LBType() == api.LBTypeLoadBalancer {
		si.AcceptProxy = c.Ingress.KeepSourceIP()
	}
	if c.Ingress.AcceptProxy() {
		si.AcceptProxy = true
	}

	userLists := make(map[string]hpi.UserList)
	var globalBasic *hpi.BasicAuth
	if c.Ingress.BasicAuthEnabled() {
		globalBasic = &hpi.BasicAuth{
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

	var globalTLS *hpi.TLSAuth
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
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonBackendInvalid,
				"spec.backend skipped, reason: %s", err,
			)
		} else if len(bk.Endpoints) == 0 {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonBackendInvalid,
				"spec.backend skipped, reason: %s", "endpoint not found",
			)
		} else {
			si.DefaultBackend = &hpi.Backend{
				BasicAuth:        bk.BasicAuth,
				Endpoints:        bk.Endpoints,
				BackendRules:     c.Ingress.Spec.Backend.BackendRules,
				RewriteRules:     c.Ingress.Spec.Backend.RewriteRules,
				HeaderRules:      c.Ingress.Spec.Backend.HeaderRules,
				Sticky:           bk.Sticky,
				StickyCookieName: bk.StickyCookieName,
				StickyCookieHash: bk.StickyCookieHash,
			}
			if c.Ingress.Spec.Backend.Name != "" {
				si.DefaultBackend.Name = c.Ingress.Spec.Backend.Name
			} else {
				si.DefaultBackend.Name = "default-backend" // TODO: Use constant
				si.DefaultBackend.NameGenerated = true
			}
			if globalBasic != nil {
				si.DefaultBackend.BasicAuth = globalBasic
			}
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
	td.TimeoutDefaults = hpi.TimeOutConfigs(c.Ingress.Timeouts())
	td.OptionsDefaults = hpi.OptionConfigs(c.Ingress.HAProxyOptions())

	if c.Ingress.Stats() {
		stats := &hpi.StatsInfo{}
		stats.Port = c.Ingress.StatsPort()
		if name := c.Ingress.StatsSecretName(); len(name) > 0 {
			secret, err := c.KubeClient.CoreV1().Secrets(c.Ingress.ObjectMeta.Namespace).Get(name, metav1.GetOptions{})
			if err == nil {
				stats.Username = string(secret.Data["username"])
				stats.PassWord = string(secret.Data["password"])
			} else {
				return errors.Errorf("failed to load stats secret for ingress %s/%s", c.Ingress.Namespace, c.Ingress.Name)
			}
		}
		td.Stats = stats
	}

	td.HTTPService = make([]*hpi.HTTPService, 0)
	td.TCPService = make([]*hpi.TCPService, 0)

	httpServices := make(map[hostBinder]*httpInfo)
	tcpServices := make(map[hostBinder]*tcpInfo)

	for ri, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			binder := hostBinder{Address: rule.HTTP.Address}
			offloadSSL := false

			if _, foundTLS := c.Ingress.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoTLS {
				offloadSSL = true
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					binder.Port = port
				} else {
					binder.Port = 443
				}
			} else {
				offloadSSL = false
				if port := rule.HTTP.Port.IntValue(); port > 0 {
					binder.Port = port
				} else {
					binder.Port = 80
				}
			}

			info := &httpInfo{Hosts: make(map[string][]*hpi.HTTPPath)}
			if v, ok := httpServices[binder]; ok {
				info = v
			} else {
				httpServices[binder] = info
			}
			info.OffloadSSL = offloadSSL

			httpPaths := info.Hosts[rule.GetHost()]
			for pi, path := range rule.HTTP.Paths {
				bk, err := c.serviceEndpoints(dnsResolvers, userLists, path.Backend.ServiceName, path.Backend.ServicePort, path.Backend.HostNames)
				if err != nil {
					c.recorder.Eventf(
						c.Ingress.ObjectReference(),
						core.EventTypeWarning,
						eventer.EventReasonBackendInvalid,
						"spec.rules[%d].http.paths[%d] skipped, reason: %s", ri, pi, err,
					)
				} else if len(bk.Endpoints) == 0 {
					c.recorder.Eventf(
						c.Ingress.ObjectReference(),
						core.EventTypeWarning,
						eventer.EventReasonBackendInvalid,
						"spec.rules[%d].http.paths[%d] skipped, reason: %s", ri, pi, "endpoint not found",
					)
				} else {
					httpPath := &hpi.HTTPPath{
						Path: path.Path,
						Backend: &hpi.Backend{
							BasicAuth:        bk.BasicAuth,
							Endpoints:        bk.Endpoints,
							BackendRules:     path.Backend.BackendRules,
							RewriteRules:     c.rewriteTarget(path.Path, path.Backend.RewriteRules),
							HeaderRules:      path.Backend.HeaderRules,
							Sticky:           bk.Sticky,
							StickyCookieName: bk.StickyCookieName,
							StickyCookieHash: bk.StickyCookieHash,
						},
					}
					if path.Backend.IngressBackend.Name != "" {
						httpPath.Backend.Name = path.Backend.IngressBackend.Name
					} else {
						httpPath.Backend.Name = getBackendName(c.Ingress, path.Backend.IngressBackend)
						httpPath.Backend.NameGenerated = true
					}
					httpPaths = append(httpPaths, httpPath)
				}
			}
			info.Hosts[rule.GetHost()] = httpPaths
		} else if rule.TCP != nil {
			binder := hostBinder{Address: rule.TCP.Address, Port: rule.TCP.Port.IntValue()}

			info := &tcpInfo{}
			if v, ok := tcpServices[binder]; ok {
				info = v
			} else {
				tcpServices[binder] = info
			}

			bk, err := c.serviceEndpoints(dnsResolvers, userLists, rule.TCP.Backend.ServiceName, rule.TCP.Backend.ServicePort, rule.TCP.Backend.HostNames)
			if err != nil {
				c.recorder.Eventf(
					c.Ingress.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonBackendInvalid,
					"spec.rules[%d].tcp skipped, reason: %s", ri, err,
				)
			} else if len(bk.Endpoints) == 0 {
				c.recorder.Eventf(
					c.Ingress.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonBackendInvalid,
					"spec.rules[%d].tcp skipped, reason: %s", ri, "endpoint not found",
				)
			} else {
				tcpHost := &hpi.TCPHost{
					Host: rule.GetHost(),
					Backend: &hpi.Backend{
						BackendRules:     rule.TCP.Backend.BackendRules,
						Endpoints:        bk.Endpoints,
						Sticky:           bk.Sticky,
						StickyCookieName: bk.StickyCookieName,
						StickyCookieHash: bk.StickyCookieHash,
					},
				}

				if rule.TCP.Backend.Name != "" {
					tcpHost.Backend.Name = rule.TCP.Backend.Name
				} else {
					tcpHost.Backend.Name = getBackendName(c.Ingress, rule.TCP.Backend)
					tcpHost.Backend.NameGenerated = true
				}

				info.Hosts = append(info.Hosts, tcpHost)
				info.ALPNOptions = parseALPNOptions(rule.TCP.ALPN) // TODO: alpn for multi-host

				if len(info.Hosts) > 0 { // for multi-host tcp, no ssl offload
					info.CertFile = ""
				} else if ref, ok := c.Ingress.FindTLSSecret(rule.Host); ok && !rule.TCP.NoTLS {
					if ref.Kind == api.ResourceKindCertificate {
						crd, err := c.VoyagerClient.VoyagerV1beta1().Certificates(c.Ingress.Namespace).Get(ref.Name, metav1.GetOptions{})
						if err == nil {
							info.CertFile = crd.SecretName() + ".pem"
						}
					} else {
						info.CertFile = ref.Name + ".pem" // Add file extension too
					}
				}
			}
		}
	}

	if c.Ingress.SSLRedirect() {
		// case: Port 443 is used in TCP mode, if port 80 is not used, redirect port 80 -> 443
		for binder, info := range tcpServices {
			if binder.Port != 443 || tcpBlocked80(binder.Address, tcpServices) || httpBlocked80(binder.Address, httpServices) {
				continue
			}
			for _, tcpHost := range info.Hosts {
				// create a HTTP rule for port 80 that redirects path `/` to 443
				addRedirectPaths(binder.Address, tcpHost.Host, []string{"/"}, httpServices)
			}
		}

		// case: Port 443 is used in HTTP mode, if port 80 is not used, redirect port 80 -> 443
		for binder, info := range httpServices {
			if binder.Port != 443 || tcpBlocked80(binder.Address, tcpServices) || httpBlocked80(binder.Address, httpServices) {
				continue
			}
			for httpHost, httpPaths := range info.Hosts {
				var paths []string
				for _, httpPath := range httpPaths {
					paths = append(paths, httpPath.Path)
				}
				addRedirectPaths(binder.Address, httpHost, paths, httpServices)
			}
		}
	}

	// Must be checked after `ssl-redirect` annotation is processed
	if len(httpServices) == 0 && // No HTTP rule used
		!tcpBlocked80(`*`, tcpServices) && // Port 80 is not used in TCP mode
		td.DefaultBackend != nil { // Default backend is provided
		httpServices[hostBinder{Address: `*`, Port: 80}] = &httpInfo{
			Hosts: map[string][]*hpi.HTTPPath{
				"": {
					{
						Path: "/",
					},
				},
			},
		}
	}

	// `ingress.kubernetes.io/force-ssl-redirect: true`, so redirect all port 80 HTTP paths to HTTPS
	if c.Ingress.ForceSSLRedirect() {
		for binder, info := range httpServices {
			if binder.Port != 80 {
				continue
			}
			for httpHost, httpPaths := range info.Hosts {
				for i := range httpPaths {
					httpPaths[i].SSLRedirect = true
				}
				info.Hosts[httpHost] = httpPaths
			}
		}
	}

	for binder, info := range httpServices {
		fr := getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, binder.Port)
		srv := &hpi.HTTPService{
			SharedInfo:    si,
			FrontendName:  getFrontendName("http", binder.Address, binder.Port),
			Address:       binder.Address,
			Port:          binder.Port,
			FrontendRules: fr.Rules,
			OffloadSSL:    info.OffloadSSL,
			Hosts:         make([]*hpi.HTTPHost, 0),
		}
		for host, paths := range info.Hosts {
			srv.Hosts = append(srv.Hosts, &hpi.HTTPHost{
				Host:  host,
				Paths: append([]*hpi.HTTPPath(nil), paths...),
			})
		}
		if globalBasic != nil {
			srv.BasicAuth = globalBasic
			srv.RemoveBackendAuth()
		} else if globalTLS != nil {
			srv.TLSAuth = globalTLS
			srv.RemoveBackendAuth()
		} else if fr.Auth != nil && fr.Auth.Basic != nil {
			srv.BasicAuth = &hpi.BasicAuth{
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

		// parse external auth
		if fr.Auth != nil && len(fr.Auth.OAuth) > 0 {
			for i := range srv.Hosts {
				for _, oauth := range fr.Auth.OAuth {
					if oauth.Host == srv.Hosts[i].Host {
						srv.Hosts[i].ExternalAuth = &hpi.ExternalAuth{
							AuthBackend: oauth.AuthBackend,
							AuthPath:    oauth.AuthPath,
							SigninPath:  oauth.SigninPath,
							Paths:       oauth.Paths,
						}
					}
				}
			}
		}

		td.HTTPService = append(td.HTTPService, srv)
	}

	for binder, info := range tcpServices {
		fr := getFrontendRulesForPort(c.Ingress.Spec.FrontendRules, binder.Port)
		srv := &hpi.TCPService{
			SharedInfo:    si,
			FrontendName:  getFrontendName("tcp", binder.Address, binder.Port),
			Address:       binder.Address,
			Port:          strconv.Itoa(binder.Port),
			FrontendRules: fr.Rules,
			ALPNOptions:   info.ALPNOptions,
			CertFile:      info.CertFile,
			Hosts:         info.Hosts,
		}

		if globalTLS != nil {
			srv.TLSAuth = globalTLS
		} else if fr.Auth != nil && fr.Auth.TLS != nil {
			htls, err := c.getTLSAuth(fr.Auth.TLS)
			if err != nil {
				return err
			}
			if htls != nil {
				srv.TLSAuth = htls
			}
		}

		td.TCPService = append(td.TCPService, srv)
	}

	td.DNSResolvers = make([]*api.DNSResolver, 0, len(dnsResolvers))
	for k := range dnsResolvers {
		td.DNSResolvers = append(td.DNSResolvers, dnsResolvers[k])
	}

	td.UserLists = make([]hpi.UserList, 0, len(userLists))
	for k := range userLists {
		td.UserLists = append(td.UserLists, userLists[k])
	}

	// assign node-ports
	if c.Ingress.UseNodePort() {
		nodePortSvc, err := c.KubeClient.CoreV1().Services(c.Ingress.GetNamespace()).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		portMapping := make(map[int32]int32)
		for _, port := range nodePortSvc.Spec.Ports {
			portMapping[port.Port] = port.NodePort
		}
		for _, svc := range td.HTTPService {
			svc.NodePort = portMapping[int32(svc.Port)]

			// check if at-least one path with ssl-redirect exists
			// if ssl-redirect-path found, check for 443-nodeport, error out if 443-nodeport not found
			if svc.Port == 80 && svc.RedirectSSL() {
				var found bool
				svc.NodePortFor443, found = portMapping[443]
				if !found {
					return errors.Errorf("failed to process ssl-redirect: nodeport for port 443 not found")
				}
			}
		}
	}

	c.logger.Debugf("Rendering haproxy.cfg for Ingress %s/%s using data: %s", c.Ingress.Namespace, c.Ingress.Name, td)
	if cfg, err := template.RenderConfig(td); err != nil {
		return err
	} else {
		c.HAProxyConfig = cfg
		c.logger.Debugf("Generated haproxy.cfg for Ingress %s/%s", c.Ingress.Namespace, c.Ingress.Name)
	}
	return nil
}

// create a HTTP rule for port 80 that redirects tlsPaths to 443
func addRedirectPaths(address, tlsHost string, tlsPaths []string, httpServices map[hostBinder]*httpInfo) {
	i80, i80Found := httpServices[hostBinder{Address: address, Port: 80}]
	if !i80Found {
		i80 = &httpInfo{
			Hosts: map[string][]*hpi.HTTPPath{
				tlsHost: make([]*hpi.HTTPPath, 0),
			},
		}
	} else {
		if _, ok := i80.Hosts[tlsHost]; !ok {
			i80.Hosts[tlsHost] = make([]*hpi.HTTPPath, 0)
		}
	}

	httpPaths := i80.Hosts[tlsHost]
	httpPathMap := make(map[string]*hpi.HTTPPath)
	for _, p := range httpPaths {
		httpPathMap[p.Path] = p
	}

	for _, tlsPath := range tlsPaths {
		if _, ok := httpPathMap[tlsPath]; !ok {
			httpPaths = append(httpPaths, &hpi.HTTPPath{
				Path:        tlsPath,
				SSLRedirect: true,
			})
		}
	}

	i80.Hosts[tlsHost] = httpPaths
	httpServices[hostBinder{Address: address, Port: 80}] = i80
}

// if HTTP mode uses port 80, so we can't setup 80 -> 443 redirection
func httpBlocked80(address string, httpServices map[hostBinder]*httpInfo) bool {
	if address == `*` {
		for b := range httpServices {
			if b.Port == 80 && b.Address != `*` {
				return true
			}
		}
		return false
	} else {
		_, ok := httpServices[hostBinder{Address: `*`, Port: 80}]
		return ok
	}
}

// if TCP mode uses port 80, so we can't setup 80 -> 443 redirection
func tcpBlocked80(address string, tcpServices map[hostBinder]*tcpInfo) bool {
	if address == `*` {
		for b := range tcpServices {
			if b.Port == 80 {
				return true
			}
		}
		return false
	} else {
		_, ok := tcpServices[hostBinder{Address: address, Port: 80}]
		return ok
	}
}

func getBasicAuthUsers(userLists map[string]hpi.UserList, sec *core.Secret) ([]string, error) {
	listNames := make([]string, 0)

	for name, data := range sec.Data {
		listName := sec.Name + "-" + name
		listNames = append(listNames, listName)

		if _, found := userLists[listName]; found {
			continue
		}

		users := make([]hpi.AuthUser, 0)
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) <= 0 {
				continue
			}
			sep := strings.Index(line, ":")
			if sep == -1 {
				return nil, errors.Errorf("missing ':' on userlist")
			}
			userName := line[0:sep]
			if userName == "" {
				return nil, errors.Errorf("missing username on userlist")
			}
			if sep == len(line)-1 || line[sep:] == "::" {
				return nil, errors.Errorf("missing '%v' password on userlist", userName)
			}
			user := hpi.AuthUser{}
			// if usr::pwd
			if string(line[sep+1]) == ":" {
				pass, err := crypt.NewFromHash(sha512_crypt.MagicPrefix).Generate([]byte(line[sep+2:]), nil)
				if err != nil {
					return nil, err
				}
				user = hpi.AuthUser{
					Username:  userName,
					Password:  pass,
					Encrypted: true,
				}
			} else {
				user = hpi.AuthUser{
					Username:  userName,
					Password:  line[sep+1:],
					Encrypted: true,
				}
			}
			users = append(users, user)
		}
		userLists[listName] = hpi.UserList{Name: listName, Users: users}
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

func getEndpointName(ep core.EndpointAddress) string {
	if ep.TargetRef != nil {
		return "pod-" + ep.TargetRef.Name
	}
	return "pod-" + ep.IP
}

func (c *controller) getServiceAuth(userLists map[string]hpi.UserList, svc *core.Service) *hpi.BasicAuth {
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
	return &hpi.BasicAuth{
		Realm:     svc.Annotations[api.AuthRealm],
		UserLists: userList,
	}
}

func (c *controller) getErrorFiles() ([]*hpi.ErrorFile, error) {
	configMap, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.ErrorFilesConfigMapName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	commands := sets.NewString("errorfile", "errorloc", "errorloc302", "errorloc303")
	codes := []string{"200", "400", "403", "405", "408", "429", "500", "502", "503", "504"}
	errorFiles := make([]*hpi.ErrorFile, 0, len(codes))
	for _, statusCode := range codes {
		if _, found := configMap.Data[statusCode+".http"]; found {
			errorFiles = append(errorFiles, &hpi.ErrorFile{
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
				return nil, errors.Errorf("found unknown errofile command %s", parts[0])
			}
			errorFiles = append(errorFiles, &hpi.ErrorFile{
				StatusCode: statusCode,
				Command:    parts[0],
				Value:      parts[1],
			})
		}
	}
	return errorFiles, nil
}

func (c *controller) getTLSAuth(cfg *api.TLSAuth) (*hpi.TLSAuth, error) {
	tlsAuthSec, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(cfg.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if _, ok := tlsAuthSec.Data["ca.crt"]; !ok {
		return nil, errors.Errorf("key ca.crt not found in TLSAuthSecret %s", tlsAuthSec.Name)
	}

	htls := &hpi.TLSAuth{
		CAFile:       cfg.SecretName + "-ca.crt",
		VerifyClient: string(cfg.VerifyClient),
		Headers:      hpi.TLSHeaders(cfg.Headers),
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

func (c *controller) convertRulesForSSLPassthrough() error {
	usesHTTPRule := false
	for i, rule := range c.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			usesHTTPRule = true

			if len(rule.HTTP.Paths) != 1 {
				return errors.Errorf("spec.rules[%d].http can't use multiple paths with %s annotation", i, api.SSLPassthrough)
			}
			if len(rule.HTTP.Paths[0].Backend.HeaderRules) != 0 {
				return errors.Errorf("spec.rules[%d].http.paths[0].backend.headerRules is not supported with %s annotation", i, api.SSLPassthrough)
			}
			if len(rule.HTTP.Paths[0].Backend.RewriteRules) != 0 {
				return errors.Errorf("spec.rules[%d].http.paths[0].backend.rewriteRules is not supported with %s annotation", i, api.SSLPassthrough)
			}

			if rule.HTTP.Port.IntValue() == 0 {
				if _, foundTLS := c.Ingress.FindTLSSecret(rule.Host); foundTLS && !rule.HTTP.NoTLS {
					rule.HTTP.Port = intstr.FromInt(443)
				} else {
					rule.HTTP.Port = intstr.FromInt(80)
				}
			}
			rule.TCP = &api.TCPIngressRuleValue{
				Address:  rule.HTTP.Address,
				Port:     rule.HTTP.Port,
				NoTLS:    rule.HTTP.NoTLS,
				NodePort: rule.HTTP.NodePort,
				Backend:  rule.HTTP.Paths[0].Backend.IngressBackend,
			}
			rule.HTTP = nil // remove http rule after conversion
			c.Ingress.Spec.Rules[i] = rule
		}
	}

	if !usesHTTPRule && c.Ingress.Spec.Backend != nil {
		if len(c.Ingress.Spec.Backend.HeaderRules) != 0 {
			return errors.Errorf("spec.backend.headerRules is not supported with %s annotation", api.SSLPassthrough)
		}
		if len(c.Ingress.Spec.Backend.RewriteRules) != 0 {
			return errors.Errorf("spec.backend.rewriteRules is not supported with %s annotation", api.SSLPassthrough)
		}
		rule := api.IngressRule{
			IngressRuleValue: api.IngressRuleValue{
				TCP: &api.TCPIngressRuleValue{
					Port:    intstr.FromInt(80),
					Backend: c.Ingress.Spec.Backend.IngressBackend,
				},
			},
		}
		c.Ingress.Spec.Rules = append(c.Ingress.Spec.Rules, rule)
		c.Ingress.Spec.Backend = nil
	}

	err := c.Ingress.IsValid(c.cfg.CloudProvider)
	if err != nil {
		return errors.Errorf("%s annotation can't be used. Reason: %v", api.SSLPassthrough, err)
	}
	return err
}
