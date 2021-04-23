/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// nolint:goconst
package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressTLS", func() {
	var (
		f      *framework.Invocation
		ing    *api.Ingress
		secret *core.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		var err error
		secret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{framework.TestDomain})
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
		if options.Cleanup {
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
			Expect(f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		}
	})

	var (
		shouldTestHttp = func(port int32, host, path string) {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())

			var httpPort core.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == port {
					httpPort = p
				}
			}

			Expect(httpPort.Port).Should(Equal(port))

			httpHost := host
			if ing.UseNodePort() {
				httpHost = host + ":" + fmt.Sprint(httpPort.NodePort)
			}

			err = f.Ingress.DoHTTP(framework.MaxRetry, httpHost, ing, f.Ingress.FilterEndpointsForPort(eps, httpPort), "GET", path, func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal(path))
			})
			Expect(err).NotTo(HaveOccurred())
		}

		shouldTestHttps = func(port int32, host, path string) {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())

			var httpsPort core.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == port {
					httpsPort = p
				}
			}

			Expect(httpsPort.Port).Should(Equal(port))

			httpsHost := host
			if ing.UseNodePort() {
				httpsHost = host + ":" + fmt.Sprint(httpsPort.NodePort)
			}

			err = f.Ingress.DoHTTPs(framework.MaxRetry, httpsHost, "", ing, f.Ingress.FilterEndpointsForPort(eps, httpsPort), "GET", path, func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal(path)) &&
					Expect(r.Host).Should(Equal(httpsHost))
			})
			Expect(err).NotTo(HaveOccurred())
		}

		shouldTestRedirect = func(host, path string) {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())

			var httpPort, httpsPort core.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == 80 {
					httpPort = p
				} else if p.Port == 443 {
					httpsPort = p
				}
			}

			Expect(httpPort.Port).Should(Equal(int32(80)))
			Expect(httpsPort.Port).Should(Equal(int32(443)))

			httpHost, httpsHost := host, host
			if ing.UseNodePort() {
				httpHost = host + ":" + fmt.Sprint(httpPort.NodePort)
				httpsHost = host + ":" + fmt.Sprint(httpsPort.NodePort)
			}

			err = f.Ingress.DoHTTPTestRedirectWithHost(framework.MaxRetry, httpHost, ing, f.Ingress.FilterEndpointsForPort(eps, httpPort), "GET", path, func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(308)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://"+httpsHost+path))
			})
			Expect(err).NotTo(HaveOccurred())
		}
	)

	Describe("Https redirect and response", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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

		It("Should redirect HTTP to HTTPs and response HTTPs", func() {
			shouldTestHttps(443, framework.TestDomain, "/testpath/ok")
			shouldTestRedirect(framework.TestDomain, "/testpath/ok")
		})
	})

	Describe("Https redirect and response for multiple hosts", func() {
		alterTestDomain := "http.voyager.test"
		var alterSecret *core.Secret

		BeforeEach(func() { // create TLS secret for alterTestDomain
			var err error
			alterSecret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{alterTestDomain})
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			Expect(f.KubeClient.CoreV1().Secrets(alterSecret.Namespace).Delete(context.TODO(), alterSecret.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
					{
						SecretName: alterSecret.Name,
						Hosts:      []string{alterTestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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
					{
						Host: alterTestDomain,
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
			By("Trying for TestDomain")
			shouldTestHttps(443, framework.TestDomain, "/testpath/ok")
			shouldTestRedirect(framework.TestDomain, "/testpath/ok")

			By("Trying for alterTestDomain")
			shouldTestHttps(443, alterTestDomain, "/testpath/ok")
			shouldTestRedirect(alterTestDomain, "/testpath/ok")
		})
	})

	Describe("Redirect with use-nodeport", func() {
		BeforeEach(func() {
			ing.Annotations[api.LBType] = api.LBTypeNodePort
			ing.Annotations[api.UseNodePort] = "true"

			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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

		It("Should redirect to HTTPs with correct nodeport", func() {
			shouldTestHttps(443, framework.TestDomain, "/testpath/ok")
			shouldTestRedirect(framework.TestDomain, "/testpath/ok")
		})
	})

	Describe("No redirect for existing path", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port:  intstr.FromInt(80),
								NoTLS: true,
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
					{
						Host: framework.TestDomain,
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

		It("should response http without redirect for existing path", func() {
			shouldTestHttp(80, framework.TestDomain, "/testpath/ok")
			shouldTestHttps(443, framework.TestDomain, "/testpath/ok")
		})
	})

	Describe("Inject new redirect path", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port:  intstr.FromInt(80),
								NoTLS: true,
								Paths: []api.HTTPIngressPath{
									{
										Path: "/alternate",
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
					{
						Host: framework.TestDomain,
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

		It("should inject new redirect path", func() {
			shouldTestHttp(80, framework.TestDomain, "/alternate/ok")
			shouldTestHttps(443, framework.TestDomain, "/testpath/ok")
			shouldTestRedirect(framework.TestDomain, "/testpath/ok")
		})
	})

	Describe("Http in port 443", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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
			shouldTestHttp(443, framework.TestDomain, "/testpath/ok")
		})
	})

	Describe("With HSTS Max Age Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.HSTSMaxAge] = "100"
			ing.Annotations[api.SSLRedirect] = "false"

			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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

			err = f.Ingress.DoHTTPs(framework.MaxRetry, framework.TestDomain, "", ing, eps, "GET", "/testpath/ok",
				func(r *client.Response) bool {
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
			ing.Annotations[api.SSLRedirect] = "false"

			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{framework.TestDomain},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
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

			err = f.Ingress.DoHTTPs(framework.MaxRetry, framework.TestDomain, "", ing, eps, "GET", "/testpath/ok",
				func(r *client.Response) bool {
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

			redirectLocation := "https://" + framework.TestDomain + "/testpath/ok"

			err = f.Ingress.DoHTTPTestRedirectWithHost(framework.MaxRetry, framework.TestDomain, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(308)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal(redirectLocation))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPTestRedirectWithHeader(framework.MaxRetry, framework.TestDomain, ing, eps, "GET", "/testpath/ok",
				map[string]string{
					"X-Forwarded-Proto": "http",
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(308)) &&
						Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
						Expect(r.ResponseHeader.Get("Location")).Should(Equal(redirectLocation))
				})
			Expect(err).NotTo(HaveOccurred())

			// should not redirect, should response normally
			err = f.Ingress.DoHTTPWithHeader(framework.MaxRetry, ing, eps, "GET", "/testpath/ok",
				map[string]string{
					"X-Forwarded-Proto": "https",
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(200)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})
			Expect(err).NotTo(HaveOccurred())

			// bad-case: without host header, just replace http with https
			// for-example: minikube: http://192.168.99.100:30001 -> https://192.168.99.100:30001
			// cloud: http://11.22.33.44:80 -> https://11.22.33.44:443
			redirectLocation = strings.Replace(eps[0], "http", "https", 1) + "/testpath/ok"
			redirectLocation = strings.Replace(redirectLocation, ":80", ":443", 1)
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(308)) &&
						Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
						Expect(r.ResponseHeader.Get("Location")).Should(Equal(redirectLocation))
				})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
