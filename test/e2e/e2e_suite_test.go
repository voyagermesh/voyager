/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"testing"
	"time"

	"voyagermesh.dev/voyager/client/clientset/versioned/scheme"
	"voyagermesh.dev/voyager/pkg/operator"
	"voyagermesh.dev/voyager/test/framework"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"gomodules.xyz/flags"
	logs "gomodules.xyz/kglog"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"kmodules.xyz/client-go/tools/clientcmd"
)

const (
	TestTimeout = 2 * time.Hour
)

var (
	root       *framework.Framework
	invocation *framework.Invocation
)

func TestE2E(t *testing.T) {
	logs.InitLogs()
	defer logs.FlushLogs()
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TestTimeout)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Voyager E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
	flags.LoggerOptions.Verbosity = "5"

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
		Expect(root.DeleteNamespace()).NotTo(HaveOccurred())
	}
})
