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
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressTCP", func() {
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

	Describe("Create TCP", func() {
		BeforeEach(func() {
			ing.Spec.TLS = []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: secret.Name,
					},
					Hosts: []string{framework.TestDomain},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4001),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
				{
					Host: framework.TestDomain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4002),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
				{
					Host: framework.TestDomain,
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4003),
							NoTLS: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
			}
		})

		It("Should test TCP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(3))
			Expect(svc.Spec.Ports[0].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))
			Expect(svc.Spec.Ports[1].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))
			Expect(svc.Spec.Ports[2].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))

			var tcpNoSSL, tcpSSL, tcpWithNoSSL core.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == 4001 {
					tcpNoSSL = p
				}

				if p.Port == 4002 {
					tcpSSL = p
				}

				if p.Port == 4003 {
					tcpWithNoSSL = p
				}
			}

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, f.Ingress.FilterEndpointsForPort(eps, tcpNoSSL), func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, f.Ingress.FilterEndpointsForPort(eps, tcpWithNoSSL), func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCPWithSSL(framework.MaxRetry, "", ing, f.Ingress.FilterEndpointsForPort(eps, tcpSSL), func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Whitelist Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.WhitelistSourceRange] = f.MinikubeIP()
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4001),
							NoTLS: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4002),
							NoTLS: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
			}
		})

		It("Should Add Whitelisted Ips to TCP Frontend", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			// Manually check if whitelisted ips are added to each tcp frontend rule of generated HAProxy config
			// TODO @ dipta: how to test if whitelist is actually working?
		})
	})

	Describe("Create TCP With Limit RPM", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4001),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
			}
			ing.Annotations[api.LimitRPM] = "2"
		})

		It("Should test TCP Connections", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4001)))

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(1, ing, eps, func(r *client.Response) bool {
				return false
			})
			Expect(err).To(HaveOccurred())

			// Wait for the rates to be reset
			time.Sleep(time.Minute * 2)

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Proxy Protocol", func() {
		Describe("With version 1", func() {
			BeforeEach(func() {
				meta, err := f.Ingress.CreateResourceWithSendProxy("v1")
				Expect(err).NotTo(HaveOccurred())
				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							TCP: &api.TCPIngressRuleValue{
								Port: intstr.FromInt(4001),
								Backend: api.IngressBackend{
									ServiceName: meta.Name,
									ServicePort: intstr.FromInt(6767),
								},
							},
						},
					},
				}
			})
			It("Should test decoded proxy-protocol header", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				svc, err := f.Ingress.GetOffShootService(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(svc.Spec.Ports)).Should(Equal(1))
				Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4001)))

				By("Checking tcp response")
				err = f.Ingress.DoTCP(framework.NoRetry, ing, eps, func(r *client.Response) bool {
					return r.Proxy.Version == 1
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With version 2", func() {
			BeforeEach(func() {
				meta, err := f.Ingress.CreateResourceWithSendProxy("v2")
				Expect(err).NotTo(HaveOccurred())
				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							TCP: &api.TCPIngressRuleValue{
								Port: intstr.FromInt(4001),
								Backend: api.IngressBackend{
									ServiceName: meta.Name,
									ServicePort: intstr.FromInt(6767),
								},
							},
						},
					},
				}
			})
			It("Should test decoded proxy-protocol header", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				svc, err := f.Ingress.GetOffShootService(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(svc.Spec.Ports)).Should(Equal(1))
				Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4001)))

				By("Checking tcp response")
				err = f.Ingress.DoTCP(framework.NoRetry, ing, eps, func(r *client.Response) bool {
					return r.Proxy.Version == 2
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
