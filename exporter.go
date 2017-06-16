package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/log"
	"github.com/appscode/pat"
	_ "github.com/appscode/voyager/client/clientset/fake"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

func NewCmdExport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export Prometheus metrics for HAProxy",
		Run: func(cmd *cobra.Command, args []string) {
			export()
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			analytics.ExporterStopped()
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&haProxyServerMetricFields, "haproxy.server-metric-fields", haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	cmd.Flags().DurationVar(&haProxyTimeout, "haproxy.timeout", haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	return cmd
}

func export() {
	if enableAnalytics {
		analytics.Enable()
	}

	log.Infoln("Starting Voyager exporter...")
	analytics.ExporterStarted()

	var err error
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
