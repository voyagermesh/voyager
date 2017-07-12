package config

type Options struct {
	CloudProvider   string
	CloudConfigFile string
	HAProxyImage    string
	IngressClass    string
	EnableRBAC      bool
}
