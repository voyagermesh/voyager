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
}

func NewConfig() *Config {
	return &Config{
		Master:                "",
		KubeConfig:            "",
		ProviderName:          "",
		ClusterName:           "",
		LoadbalancerImageName: "appscode/haproxy:1.7.2-k8s",
	}
}

func (s *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	fs.StringVarP(&s.ProviderName, "cloud-provider", "c", s.ProviderName, "Name of cloud provider")
	fs.StringVarP(&s.ClusterName, "cluster-name", "k", s.ClusterName, "Name of Kubernetes cluster")
	fs.StringVarP(&s.LoadbalancerImageName, "haproxy-image", "h", s.LoadbalancerImageName, "haproxy image name to be run")
}
