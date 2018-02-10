package e2e

import (
	"testing"
	"time"

	"github.com/appscode/go/runtime"
	"github.com/appscode/voyager/pkg/config"
	hpdata "github.com/appscode/voyager/pkg/haproxy/template"
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
	/*
	op := operator.New(
		root.KubeClient,
		root.CRDClient,
		root.VoyagerClient,
		nil,
		config.OperatorOptions{
			CloudProvider:   root.Config.CloudProviderName,
			DockerRegistry:  root.Config.DockerRegistry,
			HAProxyImageTag: root.Config.HAProxyImageTag,
			IngressClass:    root.Config.IngressClass,
			NumThreads:      1,
		},
	)

	By("Ensuring Test Namespace " + root.Config.TestNamespace)
	err := root.EnsureNamespace()
	Expect(err).NotTo(HaveOccurred())

	config.LoggerOptions.Verbosity = "5"

	if !root.Config.InCluster {
		By("Running Controller in Local mode")
		err := op.Setup()
		Expect(err).NotTo(HaveOccurred())

		err = hpdata.LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
		Expect(err).NotTo(HaveOccurred())

		//stop := make(chan struct{})
		//defer close(stop)
		go op.Run(nil)
	}
	root.EventuallyCRD().Should(Succeed())

	Eventually(invocation.Ingress.Setup).Should(BeNil())
	*/
})

var _ = AfterSuite(func() {
	if root.Config.Cleanup {
		root.DeleteNamespace()
		invocation.Ingress.Teardown()
	}
})
