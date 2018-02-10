package config

import (
	"fmt"
	"time"

	"github.com/appscode/go/log/golog"
	core "k8s.io/api/core/v1"
)

type OperatorOptions struct {
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
	MaxNumRequeues              int
	NumThreads                  int
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string
}

func (options OperatorOptions) HAProxyImage() string {
	return fmt.Sprintf("%s/haproxy:%s", options.DockerRegistry, options.HAProxyImageTag)
}

func (options OperatorOptions) ExporterImage() string {
	return fmt.Sprintf("%s/voyager:%s", options.DockerRegistry, options.ExporterImageTag)
}

func (options OperatorOptions) WatchNamespace() string {
	if options.RestrictToOperatorNamespace {
		return options.OperatorNamespace
	}
	return core.NamespaceAll
}

var (
	AnalyticsClientID string
	EnableAnalytics   = true
	LoggerOptions     golog.Options
)
