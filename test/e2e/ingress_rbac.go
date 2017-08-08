package e2e

import (
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("IngressWithRBACEnabled", func() {
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
		Expect(f.Ingress.IsTargetCreated(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
		}
	})

	Describe("With RBAC", func() {
		BeforeEach(func() {
			if !f.Config.RBACEnabled {
				Skip("RBAC is Not enabled")
			}

			if !f.Config.InCluster {
				Skip("RBAC can only be work in 'in-cluster' mode")
			}
		})

		It("Should test RBAC Support", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			_, err = f.KubeClient.CoreV1().ServiceAccounts(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.KubeClient.RbacV1beta1().Roles(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.KubeClient.RbacV1beta1().RoleBindings(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
