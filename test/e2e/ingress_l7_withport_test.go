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
	"net/http"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressWithCustomPorts", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
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
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port: intstr.FromInt(3001),
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(9090),
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

		It("Should create Loadbalancer in port 9090", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(3001)))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.ServerPort).Should(Equal(":9090"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("LBType LoadBalancer with NodePort set", func() {
		var nodePort int
		BeforeEach(func() {
			nodePort = f.Ingress.GetFreeNodePort(32700)
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NodePort: intstr.FromInt(nodePort),
								Paths: []api.HTTPIngressPath{
									{
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(9090),
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

		It("Should check update", func() {
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))
			Expect(svc.Spec.Ports[0].NodePort).Should(BeNumerically(">=", 1))
			Expect(svc.Spec.Ports[0].NodePort).Should(Equal(int32(nodePort)))
		})
	})

	Describe("LBType LoadBalancer with NodePort set on update", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(9090),
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

		It("Should check update", func() {
			if options.CloudProvider != api.ProviderMinikube {
				Skip("CloudProvider Needs to be configured for NodePort")
			}

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))
			Expect(svc.Spec.Ports[0].NodePort).Should(BeNumerically(">=", 1))

			nodePort := f.Ingress.GetFreeNodePort(32701)
			tobeUpdated, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())
			tobeUpdated.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(nodePort)
			err = f.Ingress.Update(tobeUpdated)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() int32 {
				service, err := f.Ingress.GetOffShootService(tobeUpdated)
				if err == nil {
					if len(service.Spec.Ports) > 0 {
						return service.Spec.Ports[0].NodePort
					}
				}
				return 0
			}, "5m", "10s").Should(Equal(int32(nodePort)))

		})
	})

	Describe("NodePort set", func() {
		var nodePort int
		BeforeEach(func() {
			nodePort = f.Ingress.GetFreeNodePort(32702)
			ing.Annotations[api.LBType] = api.LBTypeNodePort
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NodePort: intstr.FromInt(nodePort),
								Paths: []api.HTTPIngressPath{
									{
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(9090),
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

		It("Should check update", func() {
			if options.CloudProvider != api.ProviderMinikube {
				Skip("CloudProvider needs to be configured for NodePort")
			}

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))
			Expect(svc.Spec.Ports[0].NodePort).Should(BeNumerically(">=", 1))
			Expect(svc.Spec.Ports[0].NodePort).Should(Equal(int32(nodePort)))
		})
	})

	Describe("NodePort set on update", func() {
		BeforeEach(func() {
			ing.Annotations[api.LBType] = api.LBTypeNodePort
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(9090),
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

		It("Should check update", func() {
			if options.CloudProvider != api.ProviderMinikube {
				Skip("CloudProvider needs to be configured for NodePort")
			}

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))
			Expect(svc.Spec.Ports[0].NodePort).Should(BeNumerically(">=", 1))

			nodePort := f.Ingress.GetFreeNodePort(32705)
			tobeUpdated, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())
			tobeUpdated.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(nodePort)
			err = f.Ingress.Update(tobeUpdated)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() int32 {
				service, err := f.Ingress.GetOffShootService(tobeUpdated)
				if err == nil {
					if len(service.Spec.Ports) > 0 {
						return service.Spec.Ports[0].NodePort
					}
				}
				return 0
			}, "5m", "10s").Should(Equal(int32(nodePort)))

		})
	})
})
