package e2e

import (
	"testing"
	"time"

	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/clientcmd"
	"github.com/appscode/voyager/client/clientset/versioned/scheme"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/operator"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
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
	junitReporter := reporters.NewJUnitReporter("report.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Voyager E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	scheme.AddToScheme(clientsetscheme.Scheme)
	config.LoggerOptions.Verbosity = "5"

	options.validate()

	clientConfig, err := clientcmd.BuildConfigFromContext(options.KubeConfig, options.KubeContext)
	Expect(err).NotTo(HaveOccurred())

	operatorConfig := operator.NewOperatorConfig(clientConfig)

	err = options.ApplyTo(operatorConfig)
	Expect(err).NotTo(HaveOccurred())

	root = framework.New(operatorConfig, options.TestNamespace, options.Cleanup)

	By("Ensuring Test Namespace " + options.TestNamespace)
	err = root.EnsureNamespace()
	Expect(err).NotTo(HaveOccurred())

	invocation = root.Invoke()

	if !meta.PossiblyInCluster() {
		go root.Operator.RunInformers(nil)
		select {} // run operator locally without running tests
	}

	Eventually(invocation.Ingress.Setup).Should(BeNil())
})

var _ = AfterSuite(func() {
	if !options.Cleanup {
		return
	}
	if invocation != nil {
		invocation.Ingress.Teardown()
	}
	if root != nil {
		root.DeleteNamespace()
	}
})
