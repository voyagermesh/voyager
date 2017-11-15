package api

type AgentType string

const (
	AgentPrometheusBuiltin AgentType = "prometheus-builtin"
	AgentCoreOSPrometheus  AgentType = "coreos-prometheus-operator"
)

type AgentSpec struct {
	// Valid values: coreos-prometheus-operator
	Agent      AgentType       `json:"agent,omitempty"`
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
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
	Add(sp StatsAccessor, spec *AgentSpec) error
	Update(sp StatsAccessor, old, new *AgentSpec) error
	Delete(sp StatsAccessor, spec *AgentSpec) error
}

type StatsAccessor interface {
	GetNamespace() string
	ServiceName() string
	ServiceMonitorName() string
	Path() string
	Scheme() string
}
