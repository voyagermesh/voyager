package config

import (
	"fmt"
	"time"

	"github.com/appscode/go/log/golog"
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
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string
}

func (opt Options) HAProxyImage() string {
	return fmt.Sprintf("%s/haproxy:%s", opt.DockerRegistry, opt.HAProxyImageTag)
}

func (opt Options) ExporterImage() string {
	return fmt.Sprintf("%s/voyager:%s", opt.DockerRegistry, opt.ExporterImageTag)
}

func (opt Options) WatchNamespace() string {
	if opt.RestrictToOperatorNamespace {
		return opt.OperatorNamespace
	}
	return core.NamespaceAll
}

var (
	AnalyticsClientID string
	EnableAnalytics   = true
	LoggerOptions     golog.Options
)
