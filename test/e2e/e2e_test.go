package e2e

import (
	"testing"
	"time"

	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/operator"
	"github.com/appscode/voyager/test/testframework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

const (
	TestTimeout = 2 * time.Hour
)

var (
	root *testframework.Framework
)

func TestE2E(t *testing.T) {
	By("Initializing test Framework")
	root = testframework.New()

	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TestTimeout)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
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

	By("Ensuring Test Namespace" + root.Config.TestNamespace)
	err := root.EnsureNamespace()
	Expect(err).NotTo(HaveOccurred())

	if !root.Config.InCluster {
		By("Running Controller in non-cluster mode")
		err := controller.Setup()
		Expect(err).NotTo(HaveOccurred())
		go controller.Run()
	}
	root.EventuallyTPR().Should(Succeed())
})

var _ = AfterSuite(func() {
	if root.Config.Cleanup {
		root.DeleteNamespace()
	}
})
