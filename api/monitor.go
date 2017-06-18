package api

import "fmt"

const (
	AgentCoreosOperator = "COREOS_OPERATOR"

	MonitoringAgent              = EngressKey + "/monitoring-agent"                         // COREOS_OPERATOR
	ServiceMonitorNamespace      = EngressKey + "/service-monitor-namespace"                // Kube NS where service monitors will be created
	ServiceMonitorLabels         = EngressKey + "/service-monitor-labels"                   // map[string]string used to select Prometheus instance
	ServiceMonitorScrapeInterval = EngressKey + "/service-monitor-endpoint-scrape-interval" // scrape interval
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

func (r Ingress) MonitorSpec() (*MonitorSpec, error) {
	agent := getString(r.Annotations, MonitoringAgent)
	if agent == "" {
		return nil, nil
	}
	if agent != AgentCoreosOperator {
		return nil, fmt.Errorf("Unknown monitoring agent %s", agent)
	}

	var err error
	var prom PrometheusSpec
	prom.Namespace = getString(r.Annotations, ServiceMonitorNamespace)
	if prom.Namespace == "" {
		return nil, fmt.Errorf("Missing %s anootation", ServiceMonitorNamespace)
	}
	prom.Labels, err = getMap(r.Annotations, ServiceMonitorLabels)
	if err != nil {
		return nil, err
	}
	prom.Interval = getString(r.Annotations, ServiceMonitorScrapeInterval)
	return &MonitorSpec{Prometheus: &prom}, err
}
