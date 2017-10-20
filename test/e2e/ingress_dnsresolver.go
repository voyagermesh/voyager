package e2e

import (
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressWithDNSResolvers", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress

		svcResolveDNSWithNS,
		svcNotResolvesRedirect,
		svcResolveDNSWithoutNS *apiv1.Service
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		var err error
		svcResolveDNSWithNS = &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: f.Ingress.Namespace(),
				Annotations: map[string]string{
					api.UseDNSResolver:         "true",
					api.DNSResolverNameservers: `["8.8.8.8:53", "8.8.4.4:53"]`,
				},
			},
			Spec: apiv1.ServiceSpec{
				Type:         apiv1.ServiceTypeExternalName,
				ExternalName: "google.com",
			},
		}

		_, err = f.KubeClient.CoreV1().Services(svcResolveDNSWithNS.Namespace).Create(svcResolveDNSWithNS)
		Expect(err).NotTo(HaveOccurred())

		svcNotResolvesRedirect = &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: f.Ingress.Namespace(),
			},
			Spec: apiv1.ServiceSpec{
				Type:         apiv1.ServiceTypeExternalName,
				ExternalName: "google.com",
			},
		}

		_, err = f.KubeClient.CoreV1().Services(svcNotResolvesRedirect.Namespace).Create(svcNotResolvesRedirect)
		Expect(err).NotTo(HaveOccurred())

		svcResolveDNSWithoutNS = &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: f.Ingress.Namespace(),
			},
			Spec: apiv1.ServiceSpec{
				Type:         apiv1.ServiceTypeExternalName,
				ExternalName: "google.com",
			},
		}

		_, err = f.KubeClient.CoreV1().Services(svcResolveDNSWithoutNS.Namespace).Create(svcResolveDNSWithoutNS)
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
			f.KubeClient.CoreV1().Services(svcResolveDNSWithNS.Namespace).Delete(svcResolveDNSWithNS.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Services(svcResolveDNSWithNS.Namespace).Delete(svcResolveDNSWithNS.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Services(svcResolveDNSWithNS.Namespace).Delete(svcResolveDNSWithNS.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("ExternalNameResolver", func() {
		BeforeEach(func() {
			ing.Spec = api.IngressSpec{
				Backend: &api.HTTPIngressBackend{
					IngressBackend: api.IngressBackend{
						ServiceName: svcNotResolvesRedirect.Name,
						ServicePort: intstr.FromString("80"),
					}},
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/test-dns",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: svcResolveDNSWithNS.Name,
												ServicePort: intstr.FromString("80"),
											}},
									},
									{
										Path: "/test-no-dns",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: svcNotResolvesRedirect.Name,
												ServicePort: intstr.FromString("80"),
											}},
									},
									{
										Path: "/test-no-backend-redirect",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: svcResolveDNSWithoutNS.Name,
												ServicePort: intstr.FromString("80"),
											}},
									},
									{
										Path: "/test-no-backend-rule-redirect",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: svcNotResolvesRedirect.Name,
												ServicePort: intstr.FromString("80"),
												BackendRule: []string{
													"http-request redirect location https://google.com code 302",
												},
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
								Paths: []api.HTTPIngressPath{
									{
										Path: "/redirect-rule",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												BackendRule: []string{
													"http-request redirect location https://github.com/appscode/discuss/issues code 301",
												},
												ServiceName: svcNotResolvesRedirect.Name,
												ServicePort: intstr.FromString("80"),
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
								Paths: []api.HTTPIngressPath{
									{
										Path: "/redirect",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: svcNotResolvesRedirect.Name,
												ServicePort: intstr.FromString("80"),
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
								Paths: []api.HTTPIngressPath{
									{
										Path: "/back-end",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromString("8989"),
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

		It("Should test dns resolvers", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling /test-no-dns")
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/test-no-dns", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("http://google.com:80"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Calling /test-no-backend-redirect")
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/test-no-backend-redirect", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("http://google.com:80"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Calling /test-no-backend-rule-redirect")
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/test-no-backend-rule-redirect", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(302)) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://google.com"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Calling /test-dns")
			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/test-dns", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(404))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Calling /default")
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/default", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("http://google.com:80"))
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should test dns with backend rules", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling /redirect-rule")
			err = f.Ingress.DoHTTPTestRedirect(framework.MaxRetry, ing, eps, "GET", "/redirect-rule", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
