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
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"voyagermesh.dev/voyager/pkg/cmds/server"

	"gomodules.xyz/x/crypto/rand"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type E2EOptions struct {
	*server.OperatorOptions

	KubeContext     string
	KubeConfig      string
	TestNamespace   string
	Cleanup         bool
	TestCertificate bool
	DumpLocation    string
	LBPersistIP     string
}

var (
	options = &E2EOptions{
		OperatorOptions: server.NewOperatorOptions(),
		KubeConfig: func() string {
			kubecfg := os.Getenv("KUBECONFIG")
			if kubecfg != "" {
				return kubecfg
			}
			return filepath.Join(homedir.HomeDir(), ".kube", "config")
		}(),
		TestNamespace:   rand.WithUniqSuffix("test-voyager"),
		Cleanup:         true,
		TestCertificate: false,
		DumpLocation:    os.TempDir(),
	}
)

// xref: https://github.com/onsi/ginkgo/issues/602#issuecomment-559421839
func TestMain(m *testing.M) {
	options.AddGoFlags(flag.CommandLine)
	flag.StringVar(&options.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.StringVar(&options.TestNamespace, "namespace", "test-"+rand.Characters(5), "Run tests in this namespaces")
	flag.BoolVar(&options.Cleanup, "cleanup", options.Cleanup, "")
	flag.BoolVar(&options.TestCertificate, "cert", options.TestCertificate, "")
	flag.StringVar(&options.DumpLocation, "dump", os.TempDir(), "")
	flag.StringVar(&options.LBPersistIP, "lb-ip", options.LBPersistIP, "LoadBalancer persistent IP")
	flag.Parse()
	os.Exit(m.Run())
}

func (c *E2EOptions) validate() {
	if c.CloudProvider == "" {
		klog.Fatal("Provider name required, not provided")
	}
	if !strings.HasPrefix(c.TestNamespace, "test-") {
		klog.Fatal("Namespace is not a Test namespace")
	}
}
