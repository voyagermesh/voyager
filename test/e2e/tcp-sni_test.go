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

var _ = Describe("Ingress TCP SNI", func() {
	var (
		f              *framework.Invocation
		ing            *api.Ingress
		secret         *core.Secret
		wildcardSecret *core.Secret
		domain         = "voyager.appscode.test"
		wildcardDomain = "*.appscode.test"
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)

		// remove "ssl-verify-none" annotation from test-service
		meta := metav1.ObjectMeta{
			Name:      f.Ingress.TestServerHTTPSName(),
			Namespace: f.Namespace(),
		}
		_, _, err := core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(obj *core.Service) *core.Service {
			delete(obj.Annotations, "ingress.appscode.com/backend-tls")
			return obj
		}, metav1.PatchOptions{})
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
		// restore "ssl-verify-none" annotation in test-service
		meta := metav1.ObjectMeta{
			Name:      f.Ingress.TestServerHTTPSName(),
			Namespace: f.Namespace(),
		}
		_, _, err := core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(obj *core.Service) *core.Service {
			obj.Annotations = map[string]string{
				"ingress.appscode.com/backend-tls": "ssl verify none",
			}
			return obj
		}, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		if options.Cleanup {
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
		}
	})

	Describe("Without TLS", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: domain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerHTTPSName(),
								ServicePort: intstr.FromInt(443),
							},
						},
					},
				},
				{
					Host: wildcardDomain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerHTTPSName(),
								ServicePort: intstr.FromInt(3443),
							},
						},
					},
				},
			}
		})

		It("Should response based on Host", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8443)))

			By("Request with host: voyager.appscode.test")
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, domain, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":6443"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Request with host: http.appscode.test") // matches wildcard domain
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, "http.appscode.test", eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":3443"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With NoTLS and Spec.TLS", func() {
		BeforeEach(func() {
			var err error
			secret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{domain})
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			if options.Cleanup {
				Expect(f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			}
		})
		BeforeEach(func() {
			ing.Spec.TLS = []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: secret.Name,
					},
					Hosts: []string{domain},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: domain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							NoTLS: true,
							Port:  intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerHTTPSName(),
								ServicePort: intstr.FromInt(443),
							},
						},
					},
				},
				{
					Host: wildcardDomain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerHTTPSName(),
								ServicePort: intstr.FromInt(3443),
							},
						},
					},
				},
			}
		})

		It("Should response based on Host", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8443)))

			By("Request with host: voyager.appscode.test")
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, domain, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":6443"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Request with host: http.appscode.test") // matches wildcard domain
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, "http.appscode.test", eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":3443"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With TLS", func() {
		BeforeEach(func() {
			var err error
			secret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{domain})
			Expect(err).NotTo(HaveOccurred())
			wildcardSecret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{wildcardDomain})
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			if options.Cleanup {
				Expect(f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
				Expect(f.KubeClient.CoreV1().Secrets(wildcardSecret.Namespace).Delete(context.TODO(), wildcardSecret.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			}
		})

		BeforeEach(func() {
			ing.Spec.TLS = []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: secret.Name,
					},
					Hosts: []string{domain},
				},
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: wildcardSecret.Name,
					},
					Hosts: []string{wildcardDomain},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: domain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(8989),
							},
						},
					},
				},
				{
					Host: wildcardDomain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(8443),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(9090),
							},
						},
					},
				},
			}
		})

		It("Should response based on Host", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8443)))

			By("Request with host: voyager.appscode.test")
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, domain, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":8989"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Request with host: http.appscode.test") // matches wildcard domain
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, "http.appscode.test", eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":9090"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
