/*
Copyright The Voyager Authors.

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
package e2e

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/voyager/pkg/cmds/server"

	"k8s.io/client-go/util/homedir"
)

type E2EOptions struct {
	*server.OperatorOptions

	KubeContext        string
	KubeConfig         string
	TestNamespace      string
	Cleanup            bool
	TestCertificate    bool
	DumpLocation       string
	LBPersistIP        string
	OperatorOnly       bool
	SelfHostedOperator bool
}

var (
	options = &E2EOptions{
		OperatorOptions:    server.NewOperatorOptions(),
		KubeConfig:         filepath.Join(homedir.HomeDir(), ".kube", "config"),
		TestNamespace:      rand.WithUniqSuffix("test-voyager"),
		Cleanup:            true,
		TestCertificate:    false,
		DumpLocation:       os.TempDir(),
		OperatorOnly:       false,
		SelfHostedOperator: false,
	}
)

func init() {
	options.AddGoFlags(flag.CommandLine)
	flag.StringVar(&options.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.StringVar(&options.TestNamespace, "namespace", "test-"+rand.Characters(5), "Run tests in this namespaces")
	flag.BoolVar(&options.Cleanup, "cleanup", options.Cleanup, "")
	flag.BoolVar(&options.TestCertificate, "cert", options.TestCertificate, "")
	flag.StringVar(&options.DumpLocation, "dump", os.TempDir(), "")
	flag.StringVar(&options.LBPersistIP, "lb-ip", options.LBPersistIP, "LoadBalancer persistent IP")
	flag.BoolVar(&options.OperatorOnly, "operator-only", options.OperatorOnly, "run operator locally without running tests")
	flag.BoolVar(&options.SelfHostedOperator, "selfhosted-operator", options.SelfHostedOperator, "If true, operator runs inside cluster")
	enableLogging()
}

func enableLogging() {
	defer func() {
		logs.InitLogs()
		defer logs.FlushLogs()
	}()
	flag.Set("logtostderr", "true")
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(2)
}

func (c *E2EOptions) validate() {
	if c.CloudProvider == "" {
		log.Fatal("Provider name required, not provided")
	}
	if !strings.HasPrefix(c.TestNamespace, "test-") {
		log.Fatal("Namespace is not a Test namespace")
	}
}
