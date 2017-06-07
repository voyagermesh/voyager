package monitor

import (
	kapi "k8s.io/kubernetes/pkg/api"
)

type MonitorSpec struct {
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
}

type PrometheusSpec struct {
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

type Monitor interface {
	AddMonitor(meta kapi.ObjectMeta, spec *MonitorSpec) error
	UpdateMonitor(meta kapi.ObjectMeta, old, new *MonitorSpec) error
	DeleteMonitor(meta kapi.ObjectMeta, spec *MonitorSpec) error
}
