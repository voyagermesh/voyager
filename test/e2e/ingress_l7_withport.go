package e2e

import (
	"net/http"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
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
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
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
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(9090)))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.ServerPort).Should(Equal(":9090"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("LBType LoadBalancer with NodePort set", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NodePort: intstr.FromInt(32700),
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
			Expect(svc.Spec.Ports[0].NodePort).Should(Equal(int32(32700)))
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
			if root.Config.CloudProviderName != "minikube" {
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

			tobeUpdated, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())
			tobeUpdated.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(32701)
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
			}, "5m", "10s").Should(Equal(int32(32701)))

		})
	})

	Describe("NodePort set", func() {
		BeforeEach(func() {
			ing.Annotations[api.LBType] = api.LBTypeNodePort
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								NodePort: intstr.FromInt(32702),
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
			if root.Config.CloudProviderName != "minikube" {
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
			Expect(svc.Spec.Ports[0].NodePort).Should(Equal(int32(32702)))
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
			if root.Config.CloudProviderName != "minikube" {
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

			tobeUpdated, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())
			tobeUpdated.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(32705)
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
			}, "5m", "10s").Should(Equal(int32(32705)))

		})
	})
})
