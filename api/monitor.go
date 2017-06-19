package api

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AgentCoreosPrometheus = "coreos-prometheus-operator"
	DefaultExporterPort   = 56790

	MonitoringAgent              = EngressKey + "/monitoring-agent"                         // COREOS_PROMETHEUS
	ServiceMonitorNamespace      = EngressKey + "/service-monitor-namespace"                // Kube NS where service monitors will be created
	ServiceMonitorLabels         = EngressKey + "/service-monitor-labels"                   // map[string]string used to select Prometheus instance
	ServiceMonitorTargetPort     = EngressKey + "/service-monitor-endpoint-target-port"     // Target port on container used to expose metrics
	ServiceMonitorScrapeInterval = EngressKey + "/service-monitor-endpoint-scrape-interval" // scrape interval
)

type MonitorSpec struct {
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
}

type PrometheusSpec struct {
	// Name or number of the target port of the endpoint. Mutually exclusive with port.
	TargetPort intstr.IntOrString `json:"targetPort,omitempty"`

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
	if agent != AgentCoreosPrometheus {
		return nil, fmt.Errorf("Unknown monitoring agent %s", agent)
	}

	var err error
	var prom PrometheusSpec

	prom.Namespace = getString(r.Annotations, ServiceMonitorNamespace)
	if prom.Namespace == "" {
		return nil, fmt.Errorf("Missing %s annotation", ServiceMonitorNamespace)
	}

	prom.Labels, err = getMap(r.Annotations, ServiceMonitorLabels)
	if err != nil {
		return nil, err
	}
	if len(prom.Labels) == 0 {
		return nil, fmt.Errorf("Missing %s annotation", ServiceMonitorLabels)
	}

	port, err := getInt(r.Annotations, ServiceMonitorTargetPort)
	if err != nil {
		return nil, err
	}
	if port == 0 {
		prom.TargetPort = intstr.FromInt(DefaultExporterPort)
	} else {
		prom.TargetPort = intstr.FromInt(port)
	}

	prom.Interval = getString(r.Annotations, ServiceMonitorScrapeInterval)
	if err != nil {
		return nil, err
	}

	return &MonitorSpec{Prometheus: &prom}, err
}
