package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressWithTLSAuth", func() {
	var (
		f                   *framework.Invocation
		ing                 *api.Ingress
		tlsSecret, caSecret *apiv1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		ing.Annotations[api.SSLRedirect] = "false"
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		crt, key, err := f.CertManager.NewServerCertPair()
		Expect(err).NotTo(HaveOccurred())

		if len(f.Config.DumpLocation) > 0 {
			ioutil.WriteFile(f.Config.DumpLocation+"/server.crt", crt, os.ModePerm)
			ioutil.WriteFile(f.Config.DumpLocation+"/server.key", key, os.ModePerm)
		}

		tlsSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Data: map[string][]byte{
				apiv1.TLSCertKey:       crt,
				apiv1.TLSPrivateKeyKey: key,
			},
			Type: apiv1.SecretTypeTLS,
		}
		_, err = f.KubeClient.CoreV1().Secrets(tlsSecret.Namespace).Create(tlsSecret)
		Expect(err).NotTo(HaveOccurred())

		caSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Data: map[string][]byte{
				"ca.crt": f.CertManager.CACert(),
			},
		}
		_, err = f.KubeClient.CoreV1().Secrets(caSecret.Namespace).Create(caSecret)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		By("Creating ingress with name " + ing.GetName())
		err := f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(tlsSecret.Namespace).Delete(tlsSecret.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Secrets(caSecret.Namespace).Delete(caSecret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create Required Auth", func() {
		BeforeEach(func() {
			if f.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}
			ing.Spec = api.IngressSpec{
				FrontendRules: []api.FrontendRule{
					{
						Port: intstr.FromInt(443),
						Auth: &api.AuthOption{
							TLS: &api.TLSAuth{
								SecretName:   caSecret.Name,
								VerifyClient: api.TLSAuthVerifyRequired,
								ErrorPage:    "https://http.appscode.test/testpath/ok",
							},
						},
					},
				},
				TLS: []api.IngressTLS{
					{
						Ref: &api.LocalTypedReference{
							Kind: "Secret",
							Name: tlsSecret.Name,
						},
						Hosts: []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			ccrt, ckey, err := f.CertManager.NewClientCertPair()
			Expect(err).NotTo(HaveOccurred())

			if len(f.Config.DumpLocation) > 0 {
				ioutil.WriteFile(f.Config.DumpLocation+"/ca.crt", f.CertManager.CACert(), os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.crt", ccrt, os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.key", ckey, os.ModePerm)
			}

			resolved := false
			log.Warningln("Domain 'http.appscode.test' must resolve to ips in", eps)

			// Checking for the domain is pointing to the ips found in the endpoints
			// The IPs and domain must be in /etc/hosts file
			ips, err := net.LookupHost("http.appscode.test")
			if err != nil || len(ips) == 0 {
				Skip("Domain 'http.appscode.test' do not have endpoints")
			}

		Outer:
			for _, ep := range eps {
				vep := strings.TrimLeft(ep[:strings.LastIndex(ep, ":")], "http://")
				for _, ip := range ips {
					if vep == ip {
						resolved = true
						break Outer
					}
				}
			}

			if !resolved {
				Skip("Domain 'http.appscode.test' did not point to endpoints")
			}

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "http.appscode.test", ing, eps, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())

			// Wrong Cert
			tr := getTransportForCert(f.CertManager.CACert(), tlsSecret.Data[apiv1.TLSCertKey], tlsSecret.Data[apiv1.TLSPrivateKeyKey])
			err = f.Ingress.DoTestRedirectWithTransport(framework.NoRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(302)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			// TLS Auth
			tr = getTransportForCert(f.CertManager.CACert(), ccrt, ckey)
			err = f.Ingress.DoHTTPsWithTransport(framework.MaxRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/hello"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Create With Header", func() {
		BeforeEach(func() {
			if f.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}
			ing.Spec = api.IngressSpec{
				FrontendRules: []api.FrontendRule{
					{
						Port: intstr.FromInt(443),
						Auth: &api.AuthOption{
							TLS: &api.TLSAuth{
								SecretName:   caSecret.Name,
								VerifyClient: api.TLSAuthVerifyRequired,
								ErrorPage:    "https://http.appscode.test/testpath/ok",
								Headers: map[string]string{
									"X-SSL":               "%[ssl_fc]",
									"X-SSL-Client-Verify": "%[ssl_c_verify]",
									"X-SSL-Client-CN":     "%{+Q}[ssl_c_s_dn(cn)]",
								},
							},
						},
					},
				},
				TLS: []api.IngressTLS{
					{
						Ref: &api.LocalTypedReference{
							Kind: "Secret",
							Name: tlsSecret.Name,
						},
						Hosts: []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			ccrt, ckey, err := f.CertManager.NewClientCertPair()
			Expect(err).NotTo(HaveOccurred())

			if len(f.Config.DumpLocation) > 0 {
				ioutil.WriteFile(f.Config.DumpLocation+"/ca.crt", f.CertManager.CACert(), os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.crt", ccrt, os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.key", ckey, os.ModePerm)
			}

			resolved := false
			log.Warningln("Domain 'http.appscode.test' must resolve to ips in", eps)

			// Checking for the domain is pointing to the ips found in the endpoints
			// The IPs and domain must be in /etc/hosts file
			ips, err := net.LookupHost("http.appscode.test")
			if err != nil || len(ips) == 0 {
				Skip("Domain 'http.appscode.test' do not have endpoints")
			}

		Outer:
			for _, ep := range eps {
				vep := strings.TrimLeft(ep[:strings.LastIndex(ep, ":")], "http://")
				for _, ip := range ips {
					if vep == ip {
						resolved = true
						break Outer
					}
				}
			}

			if !resolved {
				Skip("Domain 'http.appscode.test' did not point to endpoints")
			}

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "http.appscode.test", ing, eps, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())

			// TLS Auth
			tr := getTransportForCert(f.CertManager.CACert(), ccrt, ckey)
			err = f.Ingress.DoHTTPsWithTransport(framework.MaxRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/hello")) &&
					Expect(r.RequestHeaders.Get("X-SSL-Client-CN")).Should(Equal("e2e-test")) &&
					Expect(r.RequestHeaders.Get("X-SSL-Client-Verify")).Should(Equal("0"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Create Optional Auth", func() {
		BeforeEach(func() {
			if f.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}
			ing.Spec = api.IngressSpec{
				FrontendRules: []api.FrontendRule{
					{
						Port: intstr.FromInt(443),
						Auth: &api.AuthOption{
							TLS: &api.TLSAuth{
								SecretName:   caSecret.Name,
								VerifyClient: api.TLSAuthVerifyOptional,
							},
						},
					},
				},
				TLS: []api.IngressTLS{
					{
						Ref: &api.LocalTypedReference{
							Kind: "Secret",
							Name: tlsSecret.Name,
						},
						Hosts: []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			ccrt, ckey, err := f.CertManager.NewClientCertPair()
			Expect(err).NotTo(HaveOccurred())

			if len(f.Config.DumpLocation) > 0 {
				ioutil.WriteFile(f.Config.DumpLocation+"/ca.crt", f.CertManager.CACert(), os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.crt", ccrt, os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.key", ckey, os.ModePerm)
			}

			resolved := false
			log.Warningln("Domain 'http.appscode.test' must resolve to ips in", eps)

			// Checking for the domain is pointing to the ips found in the endpoints
			// The IPs and domain must be in /etc/hosts file
			ips, err := net.LookupHost("http.appscode.test")
			if err != nil || len(ips) == 0 {
				Skip("Domain 'http.appscode.test' do not have endpoints")
			}

		Outer:
			for _, ep := range eps {
				vep := strings.TrimLeft(ep[:strings.LastIndex(ep, ":")], "http://")
				for _, ip := range ips {
					if vep == ip {
						resolved = true
						break Outer
					}
				}
			}

			if !resolved {
				Skip("Domain 'http.appscode.test' did not point to endpoints")
			}

			// No Cert but Cert is optional allow
			err = f.Ingress.DoHTTPs(framework.NoRetry, "http.appscode.test", "", ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/hello"))
			})
			Expect(err).NotTo(HaveOccurred())

			// Wrong Cert Reject
			tr := getTransportForCert(f.CertManager.CACert(), tlsSecret.Data[apiv1.TLSCertKey], tlsSecret.Data[apiv1.TLSPrivateKeyKey])
			err = f.Ingress.DoTestRedirectWithTransport(framework.NoRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return false
			})
			Expect(err).To(HaveOccurred())

			// TLS Auth
			tr = getTransportForCert(f.CertManager.CACert(), ccrt, ckey)
			err = f.Ingress.DoHTTPsWithTransport(framework.MaxRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/hello"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateAnnotationAuth", func() {
		BeforeEach(func() {
			if f.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}

			ing.Annotations[api.AuthTLSSecret] = caSecret.Name
			ing.Annotations[api.AuthTLSVerifyClient] = string(api.TLSAuthVerifyRequired)
			ing.Annotations[api.AuthTLSErrorPage] = "https://http.appscode.test/testpath/ok"

			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						Ref: &api.LocalTypedReference{
							Kind: "Secret",
							Name: tlsSecret.Name,
						},
						Hosts: []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			ccrt, ckey, err := f.CertManager.NewClientCertPair()
			Expect(err).NotTo(HaveOccurred())

			if len(f.Config.DumpLocation) > 0 {
				ioutil.WriteFile(f.Config.DumpLocation+"/ca.crt", f.CertManager.CACert(), os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.crt", ccrt, os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.key", ckey, os.ModePerm)
			}

			resolved := false
			log.Warningln("Domain 'http.appscode.test' must resolve to ips in", eps)

			// Checking for the domain is pointing to the ips found in the endpoints
			// The IPs and domain must be in /etc/hosts file
			ips, err := net.LookupHost("http.appscode.test")
			if err != nil || len(ips) == 0 {
				Skip("Domain 'http.appscode.test' do not have endpoints")
			}

		Outer:
			for _, ep := range eps {
				vep := strings.TrimLeft(ep[:strings.LastIndex(ep, ":")], "http://")
				for _, ip := range ips {
					if vep == ip {
						resolved = true
						break Outer
					}
				}
			}

			if !resolved {
				Skip("Domain 'http.appscode.test' did not point to endpoints")
			}

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "http.appscode.test", ing, eps, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())

			// Wrong Cert
			tr := getTransportForCert(f.CertManager.CACert(), tlsSecret.Data[apiv1.TLSCertKey], tlsSecret.Data[apiv1.TLSPrivateKeyKey])
			err = f.Ingress.DoTestRedirectWithTransport(framework.NoRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(302)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			// TLS Auth
			tr = getTransportForCert(f.CertManager.CACert(), ccrt, ckey)
			err = f.Ingress.DoHTTPsWithTransport(framework.MaxRetry, "http.appscode.test", tr, ing, []string{"https://http.appscode.test"}, "GET", "/testpath/hello", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/hello"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getTransportForCert(ca, ccrt, ckey []byte) *http.Transport {
	clientCert, err := tls.X509KeyPair(ccrt, ckey)
	Expect(err).NotTo(HaveOccurred())

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	return &http.Transport{
		TLSClientConfig: tlsConfig,
	}
}
