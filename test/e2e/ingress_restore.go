package e2e

import (
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Restore ingress offshoots", func() {
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

	FIt("Should restore configmap", func() {
		By("Deleting offshoot configmap")
		err := f.KubeClient.CoreV1().ConfigMaps(ing.Namespace).Delete(ing.OffshootName(), nil)
		Expect(err).NotTo(HaveOccurred())

		By("Checking configmap restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	It("Should restore service", func() {
		By("Deleting offshoot service")
		err := f.KubeClient.CoreV1().Services(ing.Namespace).Delete(ing.OffshootName(), nil)
		Expect(err).NotTo(HaveOccurred())

		By("Checking service restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	It("Should restore deployment", func() {
		By("Deleting haproxy deployment")
		err := f.KubeClient.AppsV1beta1().Deployments(ing.Namespace).Delete(ing.OffshootName(), nil)
		Expect(err).NotTo(HaveOccurred())

		By("Checking deployment restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})
})
