package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/log"
	"github.com/appscode/pat"
	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	_ "github.com/appscode/voyager/client/clientset/fake"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/watcher"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL      string
	kubeconfigPath string

	providerName    string
	cloudConfigFile string
	haProxyImage    string = "appscode/haproxy:1.7.6-3.0.0"
	ingressClass    string
	enableAnalytics bool = true

	address                   string        = fmt.Sprintf(":%d", api.DefaultExporterPort)
	haProxyServerMetricFields string        = hpe.ServerMetrics.String()
	haProxyTimeout            time.Duration = 5 * time.Second

	kubeClient clientset.Interface
	extClient  acs.ExtensionInterface
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run operator",
		PreRun: func(cmd *cobra.Command, args []string) {
			if enableAnalytics {
				analytics.Enable()
			}
			analytics.Send("operator", "started", Version)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			analytics.Send("operator", "stopped", Version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVarP(&providerName, "cloud-provider", "c", providerName, "Name of cloud provider")
	cmd.Flags().StringVar(&cloudConfigFile, "cloud-config", cloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	cmd.Flags().StringVar(&haProxyImage, "haproxy-image", haProxyImage, "haproxy image name to be run")
	cmd.Flags().StringVar(&ingressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&haProxyServerMetricFields, "haproxy.server-metric-fields", haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	cmd.Flags().DurationVar(&haProxyTimeout, "haproxy.timeout", haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	return cmd
}

func run() {
	if haProxyImage == "" {
		log.Fatalln("Missing required flag --haproxy-image")
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

	w := &watcher.Watcher{
		KubeClient:      kubeClient,
		ExtClient:       extClient,
		PromClient:      promClient,
		SyncPeriod:      time.Minute * 2,
		ProviderName:    providerName,
		CloudConfigFile: cloudConfigFile,
		HAProxyImage:    haProxyImage,
		IngressClass:    ingressClass,
	}

	log.Infoln("Starting Voyager operator...")

	// https://github.com/appscode/voyager/issues/229
	w.PurgeOffshootsWithDeprecatedLabels()

	go w.Run()

	selectedServerMetrics, err = hpe.FilterServerMetrics(haProxyServerMetricFields)
	if err != nil {
		log.Fatal(err)
	}
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
