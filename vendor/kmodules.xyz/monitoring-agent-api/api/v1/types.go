package v1

import (
	"strings"
	core "k8s.io/api/core/v1"
	"github.com/appscode/kutil"
)

type AgentType string

const (
	KeyAgent   = "monitoring.appscode.com/agent"
	KeyService = "monitoring.appscode.com/service"

	VendorPrometheus                 = "prometheus.io"
	AgentPrometheusBuiltin AgentType = VendorPrometheus + "/builtin"
	AgentCoreOSPrometheus  AgentType = VendorPrometheus + "/coreos-operator"
	// Deprecated
	DeprecatedAgentCoreOSPrometheus AgentType = "coreos-prometheus-operator"
)

func (at AgentType) Vendor() string {
	if at == DeprecatedAgentCoreOSPrometheus {
		return VendorPrometheus
	}
	return strings.SplitN(string(at), "/", 2)[0]
}

type AgentSpec struct {
	Agent      AgentType       `json:"agent,omitempty"`
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
	// Compute Resources required by the sidecar container.
	// Deprecated: Use podTemplate.spec.resources
	Resources *core.ResourceRequirements `json:"resources,omitempty"`
}

type PrometheusSpec struct {
	// Port number for the exporter side car.
	Port int32 `json:"port,omitempty"`

	// Namespace of Prometheus. Service monitors will be created in this namespace.
	Namespace string `json:"namespace,omitempty"`
	// Labels are key value pairs that is used to select Prometheus instance via ServiceMonitor labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Interval at which metrics should be scraped
	Interval string `json:"interval,omitempty"`

	// Parameters are key value pairs that are passed as flags to exporters.
	// Parameters map[string]string `json:"parameters,omitempty"`
}

type Agent interface {
	GetType() AgentType
	CreateOrUpdate(sp StatsAccessor, spec *AgentSpec) (kutil.VerbType, error)
	Delete(sp StatsAccessor) (kutil.VerbType, error)
}

type StatsAccessor interface {
	GetNamespace() string
	ServiceName() string
	ServiceMonitorName() string
	Path() string
	// Scheme is used to determine url scheme /metrics endpoint
	Scheme() string
}
