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

	KubeContext     string
	KubeConfig      string
	TestNamespace   string
	Cleanup         bool
	TestCertificate bool
	DumpLocation    string
	LBPersistIP     string
	OperatorOnly    bool
}

var (
	options = &E2EOptions{
		OperatorOptions: server.NewOperatorOptions(),
		KubeConfig:      filepath.Join(homedir.HomeDir(), ".kube", "config"),
		TestNamespace:   rand.WithUniqSuffix("test-voyager"),
		Cleanup:         true,
		TestCertificate: false,
		DumpLocation:    os.TempDir(),
		OperatorOnly:    false,
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
