package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/prometheus/haproxy_exporter/collector"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	const pidFileHelpText = `Path to HAProxy pid file.
	If provided, the standard process metrics get exported for the HAProxy
	process, prefixed with 'haproxy_process_...'. The haproxy_process exporter
	needs to have read access to files owned by the HAProxy process. Depends on
	the availability of /proc.
	https://prometheus.io/docs/instrumenting/writing_clientlibs/#process-metrics.`

	var (
		listenAddress             = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9101").String()
		metricsPath               = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		haProxyScrapeURI          = kingpin.Flag("haproxy.scrape-uri", "URI on which to scrape HAProxy.").Default("http://localhost/;csv").String()
		haProxySSLVerify          = kingpin.Flag("haproxy.ssl-verify", "Flag that enables SSL certificate verification for the scrape URI").Default("true").Bool()
		haProxyServerMetricFields = kingpin.Flag("haproxy.server-metric-fields", "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1").Default(collector.ServerMetrics.String()).String()
		haProxyTimeout            = kingpin.Flag("haproxy.timeout", "Timeout for trying to get stats from HAProxy.").Default("5s").Duration()
		haProxyPidFile            = kingpin.Flag("haproxy.pid-file", pidFileHelpText).Default("").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("haproxy_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting haproxy_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := collector.NewExporter(*haProxyScrapeURI, *haProxySSLVerify, *haProxyServerMetricFields, nil, *haProxyTimeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("haproxy_exporter"))

	if *haProxyPidFile != "" {
		procExporter := prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
			PidFn: func() (int, error) {
				content, err := ioutil.ReadFile(*haProxyPidFile)
				if err != nil {
					return 0, fmt.Errorf("can't read pid file: %s", err)
				}
				value, err := strconv.Atoi(strings.TrimSpace(string(content)))
				if err != nil {
					return 0, fmt.Errorf("can't parse pid file: %s", err)
				}
				return value, nil
			},
			Namespace: collector.Namespace,
		})
		prometheus.MustRegister(procExporter)
	}

	log.Infoln("Listening on", *listenAddress)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Haproxy Exporter</title></head>
             <body>
             <h1>Haproxy Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
