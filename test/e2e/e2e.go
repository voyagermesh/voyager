package e2e

import (
	"testing"
	"time"

	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/operator"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

const (
	TestTimeout = 2 * time.Hour
)

var (
	root       *framework.Framework
	invocation *framework.Invocation
)

func RunE2ETestSuit(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TestTimeout)

	root = framework.New()
	invocation = root.Invoke()

	junitReporter := reporters.NewJUnitReporter("report.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Voyager E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	controller := operator.New(
		root.KubeClient,
		root.VoyagerClient,
		nil,
		config.Options{
			CloudProvider: root.Config.CloudProviderName,
			HAProxyImage:  root.Config.HAProxyImageName,
			IngressClass:  root.Config.IngressClass,
		},
	)

	By("Ensuring Test Namespace " + root.Config.TestNamespace)
	err := root.EnsureNamespace()
	Expect(err).NotTo(HaveOccurred())

	if !root.Config.InCluster {
		By("Running Controller in Local mode")
		err := controller.Setup()
		Expect(err).NotTo(HaveOccurred())
		go controller.Run()
	}
	root.EventuallyTPR().Should(Succeed())

	Eventually(invocation.Ingress.Setup).Should(BeNil())
})

var _ = AfterSuite(func() {
	if root.Config.Cleanup {
		root.DeleteNamespace()
		invocation.Ingress.Teardown()
	}
})
