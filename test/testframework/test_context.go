package testframework

import (
	"flag"
	"sync"

	"github.com/appscode/errors"
	logginghandler "github.com/appscode/errors/h/log"
	"github.com/appscode/go/flags"
	"github.com/appscode/log"
)

func init() {
	InitTestFlags()
	errors.Handlers.Add(logginghandler.LogHandler{})
}

func Initialize() {
	InitTestFlags()
}

type TestContextType struct {
	KubeConfig string
	testConfig
}

type testConfig struct {
	Mode       string
	Verbose    bool
	E2EConfigs E2EConfig
}

type E2EConfig struct {
	Master                string
	KubeConfig            string
	ProviderName          string
	ClusterName           string
	LoadbalancerImageName string
	IngressClass          string
	Cleanup               bool
}

var TestContext TestContextType
var once sync.Once

func registerCommonFlags() {
	log.Infoln("Registering Test flags")
	flag.StringVar(&TestContext.Mode, "mode", "unit", "Running test mode")
	flag.BoolVar(&TestContext.Verbose, "verbose", false, "Run test in verbose mode")
}

func InitTestFlags() {
	once.Do(func() {
		registerCommonFlags()
		registerE2EFlags()
		registerLogLevel()
		flag.Parse()
	})
}

// Set LogLevel to Debug if no flag is provided
func registerLogLevel() {
	flag.Set("logtostderr", "true")
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(5)
}

func registerE2EFlags() {
	flag.StringVar(&TestContext.E2EConfigs.Master, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	flag.StringVar(&TestContext.E2EConfigs.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&TestContext.E2EConfigs.ProviderName, "cloud-provider", "", "Name of cloud provider")
	flag.StringVar(&TestContext.E2EConfigs.ClusterName, "cluster-name", "", "Name of Kubernetes cluster")
	flag.StringVar(&TestContext.E2EConfigs.LoadbalancerImageName, "haproxy-image", "appscode/haproxy:1.7.2-1.5.0", "haproxy image name to be run")
	flag.StringVar(&TestContext.E2EConfigs.IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	flag.BoolVar(&TestContext.E2EConfigs.Cleanup, "cleanup", true, "")
}
