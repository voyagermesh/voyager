package e2e

import (
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Backend rule with named service port", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		ing.Spec.Rules = []api.IngressRule{
			{
				IngressRuleValue: api.IngressRuleValue{
					HTTP: &api.HTTPIngressRuleValue{
						Paths: []api.HTTPIngressPath{
							{
								Path: "/testpath-1",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromInt(9090), // actual port
									},
								},
							},
							{
								Path: "/testpath-2",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										Name:        "backend-1",
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromString("http-2"), // named port, actual 8989
									},
								},
							},
						},
					},
				},
			},
		}
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
			f.Ingress.Delete(ing)
		}
	})

	It("Should use specified port", func() {
		By("Getting HTTP endpoints")
		eps, err := f.Ingress.GetHTTPEndpoints(ing)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(eps)).Should(BeNumerically(">=", 1))

		err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath-1",
			func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath-1")) &&
					Expect(r.ServerPort).Should(Equal(":9090"))
			})
		Expect(err).NotTo(HaveOccurred())

		err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath-2",
			func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath-2")) &&
					Expect(r.ServerPort).Should(Equal(":8989"))
			})
		Expect(err).NotTo(HaveOccurred())
	})
})
