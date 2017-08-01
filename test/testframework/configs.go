package testframework

import (
	"flag"
	"path/filepath"
	"strings"

	"github.com/appscode/errors"
	logginghandler "github.com/appscode/errors/h/log"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/flags"
	"github.com/appscode/log"
	"github.com/mitchellh/go-homedir"
)

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
}

var config E2EConfig

func init() {
	flag.StringVar(&config.Master, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	flag.StringVar(&config.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&config.CloudProviderName, "cloud-provider", "", "Name of cloud provider")
	flag.StringVar(&config.HAProxyImageName, "haproxy-image", "appscode/haproxy:1.7.6-3.1.0", "haproxy image name to be run")
	flag.StringVar(&config.IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	flag.BoolVar(&config.Cleanup, "cleanup", true, "")
	flag.BoolVar(&config.InCluster, "in-cluster", false, "")
	flag.StringVar(&config.DaemonHostName, "daemon-host-name", "", "Daemon host name to run daemon hosts")
	flag.StringVar(&config.LBPersistIP, "lb-ip", "", "LoadBalancer persistent IP")
	flag.StringVar(&config.TestNamespace, "namespace", "test-ing", "Run tests in this namespaces")
	flag.BoolVar(&config.RBACEnabled, "rbac", false, "")

	enableLogging()
	errors.Handlers.Add(logginghandler.LogHandler{})
}

func enableLogging() {
	flag.Set("logtostderr", "true")
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(4)
}

func (c *E2EConfig) validate() {
	if c.CloudProviderName == "" {
		log.Fatal("Provider name required, not provided")
	}

	if len(c.KubeConfig) == 0 {
		userHome, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		c.KubeConfig = filepath.Join(userHome, ".kube/config")
	}

	if len(c.TestNamespace) == 0 {
		c.TestNamespace = rand.WithUniqSuffix("test-voyager-")
	}

	if !strings.HasPrefix(c.TestNamespace, "test-") {
		log.Fatal("Namespace is not a Test namespace")
	}
}
