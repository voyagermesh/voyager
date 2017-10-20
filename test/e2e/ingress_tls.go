package e2e

import (
	"net/http"
	"strings"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressTLS", func() {
	var (
		f      *framework.Invocation
		ing    *api.Ingress
		secret *apiv1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		ing.Annotations[api.SSLRedirect] = "false"
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		// if f.Ingress.Config.CloudProviderName == "minikube" && !strings.HasPrefix(config.GinkgoConfig.FocusString, "IngressTLS") {
		// 	 Skip("run in minikube only when single specs running")
		// }
	})

	BeforeEach(func() {
		crt, key, err := f.CertManager.NewServerCertPair()
		Expect(err).NotTo(HaveOccurred())

		secret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Type: apiv1.SecretTypeTLS,
			Data: map[string][]byte{
				apiv1.TLSCertKey:       crt,
				apiv1.TLSPrivateKeyKey: key,
			},
		}
		_, err = f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
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
			f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		}
	})

	var (
		shouldTestRedirect = func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(2))
			Expect(svc.Spec.Ports[0].Port).To(Or(Equal(int32(80)), Equal(int32(443))))
			Expect(svc.Spec.Ports[1].Port).To(Or(Equal(int32(80)), Equal(int32(443))))

			var httpPort, httpsPort apiv1.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == 80 {
					httpPort = p
				}

				if p.Port == 443 {
					httpsPort = p
				}
			}

			err = f.Ingress.DoHTTPsTestRedirect(framework.MaxRetry, "http.appscode.test", ing, f.Ingress.FilterEndpointsForPort(eps, httpPort), "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPs(framework.MaxRetry, "http.appscode.test", "", ing, f.Ingress.FilterEndpointsForPort(eps, httpsPort), "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal("http.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())
		}
	)

	Describe("Https response", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
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

		It("Should response HTTPs", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))

			err = f.Ingress.DoHTTPs(framework.MaxRetry, "http.appscode.test", "", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal("http.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Https redirect port specified", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port:  intstr.FromInt(80),
								NoTLS: true,
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												BackendRule: []string{
													"redirect scheme https code 301 if !{ ssl_fc }",
												},
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port: intstr.FromInt(443),
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

		It("Should redirect HTTP", shouldTestRedirect)
	})

	Describe("Https redirect port not specified", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.test",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NoTLS: true,
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												BackendRule: []string{
													"redirect scheme https code 301 if !{ ssl_fc }",
												},
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
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

		It("Should redirect HTTP", shouldTestRedirect)
	})

	Describe("Http in port 443", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"443-with-out-ssl.test.com"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "443-with-out-ssl.test.com",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NoTLS: true,
								Port:  intstr.FromInt(443),
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

		It("Should response HTTP from port 443", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "443-with-out-ssl.test.com", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal("443-with-out-ssl.test.com"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SSL Passthrough", func() {
		BeforeEach(func() {
			ing.Annotations[api.SSLPassthrough] = "true"
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
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

		It("Should Open 443 with HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "http.appscode.test", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal("http.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With HSTS Max Age Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.HSTSMaxAge] = "100"
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
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

		It("Should Set max-age Header to Specified Value", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))

			err = f.Ingress.DoHTTPs(framework.MaxRetry, "http.appscode.test", "", ing, eps, "GET", "/testpath/ok",
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok")) &&
						Expect(r.ResponseHeader.Get("Strict-Transport-Security")).Should(Equal("max-age=100"))
				})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With HSTS Preload and Subdomains", func() {
		BeforeEach(func() {
			ing.Annotations[api.HSTSPreload] = "true"
			ing.Annotations[api.HSTSIncludeSubDomains] = "true"
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.test"},
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

		It("Should Add HSTS preload and includeSubDomains Header", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))

			err = f.Ingress.DoHTTPs(framework.MaxRetry, "http.appscode.test", "", ing, eps, "GET", "/testpath/ok",
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok")) &&
						Expect(r.ResponseHeader.Get("Strict-Transport-Security")).
							Should(Equal("max-age=15768000; preload; includeSubDomains"))
				})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Force SSL Redirect", func() {
		BeforeEach(func() {
			ing.Annotations[api.ForceSSLRedirect] = "true"
		})

		It("Should redirect HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			httpsAddr := strings.Replace(eps[0], "http", "https", 1) + "/testpath/ok"

			err = f.Ingress.DoHTTPTestRedirect(framework.NoRetry, ing, eps, "GET", "/testpath/ok",
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(301)) &&
						Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
						Expect(r.ResponseHeader.Get("Location")).Should(Equal(httpsAddr))
				})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPTestRedirectWithHeader(framework.NoRetry, ing, eps, "GET", "/testpath/ok",
				map[string]string{
					"X-Forwarded-Proto": "http",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(301)) &&
						Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
						Expect(r.ResponseHeader.Get("Location")).Should(Equal(httpsAddr))
				})
			Expect(err).NotTo(HaveOccurred())

			// should not redirect, should response normally
			err = f.Ingress.DoHTTPWithHeader(framework.NoRetry, ing, eps, "GET", "/testpath/ok",
				map[string]string{
					"X-Forwarded-Proto": "https",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(200)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
