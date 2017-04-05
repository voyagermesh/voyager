package options

import (
	"github.com/spf13/pflag"
)

type Config struct {
	Master                string
	KubeConfig            string
	ProviderName          string
	ClusterName           string
	LoadbalancerImageName string
	IngressClass          string
}

func NewConfig() *Config {
	return &Config{
		Master:                "",
		KubeConfig:            "",
		ProviderName:          "",
		ClusterName:           "",
		LoadbalancerImageName: "appscode/haproxy:1.7.2-1.5.1",
	}
}

func (s *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	fs.StringVarP(&s.ProviderName, "cloud-provider", "c", s.ProviderName, "Name of cloud provider")
	fs.StringVarP(&s.ClusterName, "cluster-name", "k", s.ClusterName, "Name of Kubernetes cluster")
	fs.StringVarP(&s.LoadbalancerImageName, "haproxy-image", "h", s.LoadbalancerImageName, "haproxy image name to be run")

	fs.StringVar(&s.IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
}
