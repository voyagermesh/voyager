package config

import (
	"time"

	apiv1 "k8s.io/api/core/v1"
)

type Options struct {
	CloudProvider               string
	CloudConfigFile             string
	HAProxyImage                string
	IngressClass                string
	EnableRBAC                  bool
	OperatorNamespace           string
	OperatorService             string
	RestrictToOperatorNamespace bool
	QPS                         float32
	Burst                       int
	ResyncPeriod                time.Duration
}

func (opt Options) WatchNamespace() string {
	if opt.RestrictToOperatorNamespace {
		return opt.OperatorNamespace
	}
	return apiv1.NamespaceAll
}
