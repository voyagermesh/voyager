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

package e2e_test

import (
	"context"
	"net/http"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	core_util "kmodules.xyz/client-go/core/v1"
)

var _ = Describe("IngressWithBasicAuth", func() {
	var (
		f           *framework.Invocation
		ing         *api.Ingress
		secret, sec *core.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		secret = &core.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			StringData: map[string]string{
				"auth": `foo::bar
				jane:E5BrlrQ5IXYK2`,

				"auth2": `auth2-foo::bar
				auth2-jane:E5BrlrQ5IXYK2`,
			},
		}
		_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
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

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}
			ing.Spec.Rules = []api.IngressRule{
				{
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
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateWithFrontendRules", func() {
		BeforeEach(func() {
			ing.Spec.FrontendRules = []api.FrontendRule{
				{
					Port: intstr.FromInt(80),
					Auth: &api.AuthOption{
						Basic: &api.BasicAuth{
							SecretName: secret.Name,
							Realm:      "Realm returned",
						},
					},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
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
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateWithDifferentFrontendRules", func() {
		BeforeEach(func() {
			sec = &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				StringData: map[string]string{
					"auth": `foo::bar-from-secret-frontend`,
				},
			}
			_, err := f.KubeClient.CoreV1().Secrets(sec.Namespace).Create(context.TODO(), sec, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if options.Cleanup {
				Expect(f.KubeClient.CoreV1().Secrets(sec.Namespace).Delete(context.TODO(), sec.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			}
		})

		BeforeEach(func() {
			ing.Spec.FrontendRules = []api.FrontendRule{
				{
					Port: intstr.FromInt(80),
					Auth: &api.AuthOption{
						Basic: &api.BasicAuth{
							SecretName: secret.Name,
							Realm:      "Realm returned",
						},
					},
				},
				{
					Port: intstr.FromInt(9090),
					Auth: &api.AuthOption{
						Basic: &api.BasicAuth{
							SecretName: sec.Name,
							Realm:      "Realm returned",
						},
					},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
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

					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Port: intstr.FromInt(9090),
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
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			var port80, port9090 core.ServicePort
			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range svc.Spec.Ports {
				if p.Port == 80 {
					port80 = p
				}

				if p.Port == 9090 {
					port9090 = p
				}
			}

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			// Call The Second HTTP Port
			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port9090),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhci1mcm9tLXNlY3JldC1mcm9udGVuZA==",
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			// Test passing valid password to the other port fails
			err = f.Ingress.DoHTTPStatusWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhci1mcm9tLXNlY3JldC1mcm9udGVuZA==",
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
				})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateAnnotationAndFrontendRules", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}
			ing.Spec.FrontendRules = []api.FrontendRule{
				{
					Port: intstr.FromInt(80),
					Auth: &api.AuthOption{
						Basic: &api.BasicAuth{
							SecretName: secret.Name,
							Realm:      "Realm returned",
						},
					},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
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

					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Port: intstr.FromInt(9090),
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
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			var port80, port9090 core.ServicePort
			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range svc.Spec.Ports {
				if p.Port == 80 {
					port80 = p
				}

				if p.Port == 9090 {
					port9090 = p
				}
			}

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port80),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			// Call The Second HTTP Port
			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				f.Ingress.FilterEndpointsForPort(eps, port9090),
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Service Auth", func() {
		BeforeEach(func() {
			meta, err := f.Ingress.CreateResourceWithServiceAuth(secret)
			Expect(err).NotTo(HaveOccurred())

			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: meta.Name,
											ServicePort: intstr.FromInt(80),
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

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPStatusWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic wrongPass",
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Service Auth Update", func() {
		var (
			secretNew *core.Secret
			meta      metav1.ObjectMeta
		)
		BeforeEach(func() {
			By("Creating new secret") // will be used when service updated
			secretNew = &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				StringData: map[string]string{
					"auth": "foo::new-bar",
				},
			}
			_, err := f.KubeClient.CoreV1().Secrets(secretNew.Namespace).Create(context.TODO(), secretNew, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// at first service will point `secret` and after update it will point `secretNew`
			meta, err = f.Ingress.CreateResourceWithServiceAuth(secret)
			Expect(err).NotTo(HaveOccurred())

			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: meta.Name,
											ServicePort: intstr.FromInt(80),
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
			By("Deleting new secret")
			err := f.KubeClient.CoreV1().Secrets(secretNew.Namespace).Delete(context.TODO(), secretNew.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should response HTTP on service update", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("Updating service annotations")
			_, _, err = core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(in *core.Service) *core.Service {
				in.Annotations[api.AuthSecret] = secretNew.Name
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for operator to process service update")
			time.Sleep(5 * time.Second)

			By("Sending request with updated auth")
			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOm5ldy1iYXI=", // foo:new-bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("Removing service annotations")
			_, _, err = core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(in *core.Service) *core.Service {
				delete(in.Annotations, api.AuthType)
				delete(in.Annotations, api.AuthRealm)
				delete(in.Annotations, api.AuthSecret)
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for operator to process service update")
			time.Sleep(5 * time.Second)

			By("Sending request without auth")
			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Both Global and Service Auth", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}

			secret2 := &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				StringData: map[string]string{
					"auth3": `auth3-foo::bar`,
				},
			}
			_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			meta, err := f.Ingress.CreateResourceWithServiceAuth(secret2)
			Expect(err).NotTo(HaveOccurred())

			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: meta.Name,
											ServicePort: intstr.FromInt(80),
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

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			// should Unauthorized, since 'secret2' will be replaced by global 'secret'
			err = f.Ingress.DoHTTPStatusWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDMtZm9vOmJhcg==", // auth3-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==", // auth2-foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==", // jane:guest
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Secret Update", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}
			ing.Spec.Rules = []api.IngressRule{
				{
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
			}
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==", // foo:bar
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			sec, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			sec.Data["auth3"] = []byte(`foo3::bar3`)
			_, err = f.KubeClient.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), sec, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for update to be done
			time.Sleep(time.Second * 30)

			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vMzpiYXIz", // foo3:bar3
				},
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
