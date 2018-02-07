package e2e

import (
	"fmt"
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Frontend rule using specified backend", func() {
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
								Path: "/testpath-0",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromInt(80),
										BackendRules: []string{
											"http-response set-header X-Ingress-Test-Header backend-0",
										},
									},
								},
							},
							{
								Path: "/testpath-1",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										Name:        "backend-1",
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromInt(80),
										BackendRules: []string{
											"http-response set-header X-Ingress-Test-Header backend-1",
										},
									},
								},
							},
							{
								Path: "/testpath-2",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										// intentionally duplicate with generated backend-0 name
										Name:        f.Ingress.TestServerName() + "." + f.Namespace() + ":80",
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromInt(80),
										BackendRules: []string{
											"http-response set-header X-Ingress-Test-Header backend-2",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ing.Spec.FrontendRules = []api.FrontendRule{
			{
				Port: intstr.FromInt(80),
				Rules: []string{
					"acl acl_testpath_3 path_beg /testpath-3",
					"acl acl_testpath_4 path_beg /testpath-4",
					"use_backend " + ing.Spec.Rules[0].HTTP.Paths[1].Backend.Name + " if acl_testpath_3",
					"use_backend " + ing.Spec.Rules[0].HTTP.Paths[2].Backend.Name + " if acl_testpath_4",
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
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
		}
	})

	It("Should use specified backend", func() {
		By("Getting HTTP endpoints")
		eps, err := f.Ingress.GetHTTPEndpoints(ing)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(eps)).Should(BeNumerically(">=", 1))

		for i := 0; i < 5; i++ {
			path := fmt.Sprintf("/testpath-%d", i)
			var header string
			if i == 0 {
				header = "backend-0"
			} else if i == 1 || i == 3 {
				header = "backend-1"
			} else if i == 2 || i == 4 {
				header = "backend-2"
			}
			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", path,
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal(path)) &&
						Expect(r.ResponseHeader.Get("X-Ingress-Test-Header")).Should(Equal(header))
				})
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
