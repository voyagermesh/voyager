package framework

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	logs "github.com/appscode/go/log/golog"
	"k8s.io/client-go/util/homedir"
)

func init() {
	flag.StringVar(&testConfigs.Master, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	flag.StringVar(&testConfigs.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&testConfigs.CloudProviderName, "cloud-provider", "", "Name of cloud provider")
	flag.StringVar(&testConfigs.HAProxyImageName, "haproxy-image", "appscode/haproxy:1.7.9-5.0.0", "haproxy image name to be run")
	flag.StringVar(&testConfigs.IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	flag.BoolVar(&testConfigs.Cleanup, "cleanup", true, "")
	flag.BoolVar(&testConfigs.InCluster, "in-cluster", false, "")
	flag.StringVar(&testConfigs.DaemonHostName, "daemon-host-name", "", "Daemon host name to run daemon hosts")
	flag.StringVar(&testConfigs.LBPersistIP, "lb-ip", "", "LoadBalancer persistent IP")
	flag.StringVar(&testConfigs.TestNamespace, "namespace", "test-"+rand.Characters(5), "Run tests in this namespaces")
	flag.BoolVar(&testConfigs.RBACEnabled, "rbac", false, "")
	flag.BoolVar(&testConfigs.TestCertificate, "cert", false, "")
	flag.StringVar(&testConfigs.DumpLocation, "dump", os.TempDir(), "")

	enableLogging()
}

type E2EConfig struct {
	Master            string
	KubeConfig        string
	CloudProviderName string
	HAProxyImageName  string
	TestNamespace     string
	IngressClass      string
	InCluster         bool
	Cleanup           bool
	DaemonHostName    string
	LBPersistIP       string
	RBACEnabled       bool
	TestCertificate   bool
	DumpLocation      string
}

var testConfigs E2EConfig

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

func (c *E2EConfig) validate() {
	if c.CloudProviderName == "" {
		log.Fatal("Provider name required, not provided")
	}

	if len(c.KubeConfig) == 0 {
		c.KubeConfig = filepath.Join(homedir.HomeDir(), ".kube/config")
	}

	if len(c.TestNamespace) == 0 {
		c.TestNamespace = rand.WithUniqSuffix("test-voyager")
	}

	if !strings.HasPrefix(c.TestNamespace, "test-") {
		log.Fatal("Namespace is not a Test namespace")
	}
}
