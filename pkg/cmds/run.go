package cmds

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/log"
	"github.com/appscode/pat"
	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/operator"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL      string
	kubeconfigPath string
	opt            config.Options = config.Options{
		HAProxyImage:      "appscode/haproxy:1.7.6-3.1.0",
		OperatorNamespace: namespace(),
		OperatorService:   "voyager-operator",
		HTTPChallengePort: 56791,
		EnableRBAC:        false,
		SyncPeriod:        30 * time.Second,
	}
	enableAnalytics bool = true

	address                   string        = fmt.Sprintf(":%d", api.DefaultExporterPortNumber)
	haProxyServerMetricFields string        = hpe.ServerMetrics.String()
	haProxyTimeout            time.Duration = 5 * time.Second

	kubeClient clientset.Interface
	extClient  acs.ExtensionInterface
)

func NewCmdRun(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Run operator",
		DisableAutoGenTag: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			if enableAnalytics {
				analytics.Enable()
			}
			analytics.Send("operator", "started", version)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			analytics.Send("operator", "stopped", version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVarP(&opt.CloudProvider, "cloud-provider", "c", opt.CloudProvider, "Name of cloud provider")
	cmd.Flags().StringVar(&opt.CloudConfigFile, "cloud-config", opt.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	cmd.Flags().StringVar(&opt.HAProxyImage, "haproxy-image", opt.HAProxyImage, "haproxy image name to be run")
	cmd.Flags().StringVar(&opt.IngressClass, "ingress-class", opt.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	cmd.Flags().BoolVar(&opt.EnableRBAC, "rbac", opt.EnableRBAC, "Enable RBAC for operator & offshoot Kubernetes objects")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	cmd.Flags().StringVar(&opt.OperatorService, "operator-service", opt.OperatorService, "Name of service used to expose voyager operator")
	cmd.Flags().IntVar(&opt.HTTPChallengePort, "http-challenge-port", opt.HTTPChallengePort, "Port used to answer ACME HTTP challenge")
	cmd.Flags().DurationVar(&opt.SyncPeriod, "resync-period", opt.SyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&haProxyServerMetricFields, "haproxy.server-metric-fields", haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	cmd.Flags().DurationVar(&haProxyTimeout, "haproxy.timeout", haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	return cmd
}

func run() {
	if opt.HAProxyImage == "" {
		log.Fatalln("Missing required flag --haproxy-image")
	}
	if opt.CloudProvider == "$CLOUD_PROVIDER" {
		log.Fatalln("Invalid cloud provider `--cloud-provider=$CLOUD_PROVIDER`")
	}
	if opt.CloudConfigFile == "$CLOUD_CONFIG" {
		log.Fatalln("Invalid cloud config file `--cloud-config=$CLOUD_CONFIG`")
	}
	if opt.IngressClass == "$INGRESS_CLASS" {
		log.Fatalln("Invalid ingress class `--ingress-class=$INGRESS_CLASS`")
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	kubeClient = clientset.NewForConfigOrDie(config)
	extClient = acs.NewForConfigOrDie(config)
	promClient, err := pcm.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	log.Infoln("Starting Voyager operator...")

	w := operator.New(kubeClient, extClient, promClient, opt)
	err = w.Setup()
	if err != nil {
		log.Fatalln(err)
	}
	// https://github.com/appscode/voyager/issues/346
	err = w.ValidateIngress()
	if err != nil {
		log.Fatalln(err)
	}
	// https://github.com/appscode/voyager/issues/229
	w.PurgeOffshootsWithDeprecatedLabels()
	go w.Run()

	m := pat.New()
	m.Get("/metrics", promhttp.Handler())
	pattern := fmt.Sprintf("/%s/v1beta1/namespaces/%s/ingresses/%s/metrics", PathParamAPIGroup, PathParamNamespace, PathParamName)
	log.Infof("URL pattern: %s", pattern)
	m.Get(pattern, http.HandlerFunc(ExportMetrics))
	m.Del(pattern, http.HandlerFunc(DeleteRegistry))
	http.Handle("/", m)
	log.Infoln("Listening on", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func namespace() string {
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return apiv1.NamespaceDefault
}
