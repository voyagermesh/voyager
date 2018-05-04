package e2e

import (
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Ingress TCP SNI", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
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
		_, _, err := core_util.CreateOrPatchService(f.KubeClient, meta, func(obj *core.Service) *core.Service {
			delete(obj.Annotations, "ingress.appscode.com/backend-tls")
			return obj
		})
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
		_, _, err := core_util.CreateOrPatchService(f.KubeClient, meta, func(obj *core.Service) *core.Service {
			obj.Annotations = map[string]string{
				"ingress.appscode.com/backend-tls": "ssl verify none",
			}
			return obj
		})
		Expect(err).NotTo(HaveOccurred())

		if options.Cleanup {
			f.Ingress.Delete(ing)
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: "http.appscode.test",
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
					Host: "ssl.appscode.test",
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

			By("Request with host: http.appscode.test")
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, "http.appscode.test", eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":6443"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Request with host: ssl.appscode.test")
			err = f.Ingress.DoHTTPWithSNI(framework.MaxRetry, "ssl.appscode.test", eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":3443"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
