package v1beta1

import (
	"fmt"
)

const (
	AgentCoreosPrometheus     = "coreos-prometheus-operator"
	StatsPortName             = "stats"
	ExporterPortName          = "http"
	DefaultExporterPortNumber = 56790

	MonitoringAgent              = EngressKey + "/monitoring-agent"                         // Name of monitoring agent
	ServiceMonitorNamespace      = EngressKey + "/service-monitor-namespace"                // Kube NS where service monitors will be created
	ServiceMonitorLabels         = EngressKey + "/service-monitor-labels"                   // map[string]string used to select Prometheus instance
	ServiceMonitorPort           = EngressKey + "/service-monitor-endpoint-port"            // Port on stats service used to expose metrics
	ServiceMonitorScrapeInterval = EngressKey + "/service-monitor-endpoint-scrape-interval" // scrape interval
)

// +k8s:openapi-gen=false
type MonitorSpec struct {
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
}

// +k8s:openapi-gen=false
type PrometheusSpec struct {
	// Port number for the exporter side car.
	Port int `json:"port,omitempty"`

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

func (r Ingress) MonitorSpec() (*MonitorSpec, error) {
	agent := GetString(r.Annotations, MonitoringAgent)
	if agent == "" {
		return nil, nil
	}
	if agent != AgentCoreosPrometheus {
		return nil, fmt.Errorf("Unknown monitoring agent %s", agent)
	}

	var err error
	var prom PrometheusSpec

	prom.Namespace = GetString(r.Annotations, ServiceMonitorNamespace)
	if prom.Namespace == "" {
		return nil, fmt.Errorf("Missing %s annotation", ServiceMonitorNamespace)
	}

	prom.Labels, err = GetMap(r.Annotations, ServiceMonitorLabels)
	if err != nil {
		return nil, err
	}
	if len(prom.Labels) <= 0 {
		return nil, fmt.Errorf("Missing %s annotation", ServiceMonitorLabels)
	}

	port, err := GetInt(r.Annotations, ServiceMonitorPort)
	if err != nil {
		return nil, err
	}
	if port == 0 {
		prom.Port = DefaultExporterPortNumber
	} else {
		prom.Port = port
	}

	prom.Interval = GetString(r.Annotations, ServiceMonitorScrapeInterval)

	return &MonitorSpec{Prometheus: &prom}, nil
}
