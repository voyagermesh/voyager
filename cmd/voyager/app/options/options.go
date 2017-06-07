package options

import (
	"time"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/spf13/pflag"
)

type Config struct {
	Master                    string
	KubeConfig                string
	ProviderName              string
	ClusterName               string
	LoadbalancerImageName     string
	IngressClass              string
	EnableAnalytics           bool
	Address                   string
	haProxyServerMetricFields string
	haProxyTimeout            time.Duration
}

func NewConfig() *Config {
	return &Config{
		Master:                "",
		KubeConfig:            "",
		ProviderName:          "",
		ClusterName:           "",
		LoadbalancerImageName: "appscode/haproxy:1.7.5-1.5.5",
		EnableAnalytics:       true,
		Address:               ":8080",
	}
}

func (s *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	fs.StringVarP(&s.ProviderName, "cloud-provider", "c", s.ProviderName, "Name of cloud provider")
	fs.StringVarP(&s.ClusterName, "cluster-name", "k", s.ClusterName, "Name of Kubernetes cluster")
	fs.StringVarP(&s.LoadbalancerImageName, "haproxy-image", "h", s.LoadbalancerImageName, "haproxy image name to be run")

	fs.StringVar(&s.IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")

	fs.BoolVar(&s.EnableAnalytics, "analytics", s.EnableAnalytics, "Send analytical event to Google Analytics")

	fs.StringVar(&s.Address, "address", s.Address, "Address to listen on for web interface and telemetry.")
	fs.StringVar(&s.haProxyServerMetricFields, "haproxy.server-metric-fields", hpe.ServerMetrics.String(), "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	fs.DurationVar(&s.haProxyTimeout, "haproxy.timeout", 5*time.Second, "Timeout for trying to get stats from HAProxy.")
}
