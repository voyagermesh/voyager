package e2e

import (
	"net/http"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
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
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		secret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Type: apiv1.SecretTypeTLS,
			StringData: map[string]string{
				"tls.key": fakeHTTPAppsCodeDevKey,
				"tls.crt": fakeHTTPAppsCodeDevCert,
			},
		}
		_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		By("Creating ingress with name " + ing.GetName())
		err := f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExists(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Https response", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{f.Ingress.TLSHostName()},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: f.Ingress.TLSHostName(),
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

			if f.Ingress.Config.CloudProviderName == "minikube" {
				ing.Spec.Rules[0].HTTP.NodePort = intstr.FromString(f.Ingress.TLSNodePortForMiniKube())
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

			err = f.Ingress.DoHTTPs(framework.MaxRetry, f.Ingress.TLSHostName(), "", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal(f.Ingress.TLSHostName()))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Https redirect", func() {
		BeforeEach(func() {
			if f.Ingress.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
				f.Ingress.Mutex.Lock()
			}

			ing.Spec = api.IngressSpec{
				TLS: []api.IngressTLS{
					{
						SecretName: secret.Name,
						Hosts:      []string{"http.appscode.dev"},
					},
				},
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.dev",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port:  intstr.FromInt(80),
								NoSSL: true,
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
						Host: "http.appscode.dev",
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

		AfterEach(func() {
			if f.Ingress.Config.CloudProviderName == "minikube" {
				f.Ingress.Mutex.Unlock()
			}
		})

		It("Should redirect HTTP", func() {
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

			err = f.Ingress.DoHTTPsTestRedirect(framework.MaxRetry, "http.appscode.dev", ing, f.Ingress.FilterEndpointsForPort(eps, httpPort), "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.dev/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPs(framework.MaxRetry, "http.appscode.dev", "", ing, f.Ingress.FilterEndpointsForPort(eps, httpsPort), "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal("http.appscode.dev"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
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
								NoSSL: true,
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
})
