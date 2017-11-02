package config

import (
	"time"

	core "k8s.io/api/core/v1"
)

type Options struct {
	CloudProvider               string
	CloudConfigFile             string
	IngressClass                string
	EnableRBAC                  bool
	OperatorNamespace           string
	OperatorService             string
	RestrictToOperatorNamespace bool
	QPS                         float32
	Burst                       int
	ResyncPeriod                time.Duration
	HAProxyImage                string
	ExporterSidecarImage        string
}

func (opt Options) WatchNamespace() string {
	if opt.RestrictToOperatorNamespace {
		return opt.OperatorNamespace
	}
	return core.NamespaceAll
}
