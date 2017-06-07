package options

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/pflag"
	kapi "k8s.io/kubernetes/pkg/api"
)

type Config struct {
	Master                string
	KubeConfig            string
	ProviderName          string
	ClusterName           string
	LoadbalancerImageName string
	IngressClass          string
	EnableAnalytics       bool
	// Exporter namespace
	ExporterNamespace string
	// Tag of Exporter
	ExporterTag string
}

func NewConfig() *Config {
	return &Config{
		Master:                "",
		KubeConfig:            "",
		ProviderName:          "",
		ClusterName:           "",
		LoadbalancerImageName: "appscode/haproxy:1.7.5-1.5.5",
		EnableAnalytics:       true,
		ExporterNamespace:     namespace(),
		ExporterTag:           "v0.8.0",
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

	// exporter flags
	fs.StringVar(&s.ExporterNamespace, "exporter.namespace", s.ExporterNamespace, "Namespace for monitoring exporter")
	fs.StringVar(&s.ExporterTag, "exporter.tag", s.ExporterTag, "Tag of monitoring exporter")
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
	return kapi.NamespaceDefault
}
