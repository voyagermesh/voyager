package monitoring

import (
	"fmt"

	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/monitoring/api"
	"path"
)

const (
	monitoringAgent              = "monitoring-agent"                         // Name of monitoring agent
	serviceMonitorNamespace      = "service-monitor-namespace"                // Kube NS where service monitors will be created
	serviceMonitorLabels         = "service-monitor-labels"                   // map[string]string used to select Prometheus instance
	serviceMonitorPort           = "service-monitor-endpoint-port"            // Port on stats service used to expose metrics
	serviceMonitorScrapeInterval = "service-monitor-endpoint-scrape-interval" // scrape interval
)

func Parse(annotations map[string]string, keyPrefix string, defaultExporterPort int32) (*api.AgentSpec, error) {
	name := meta.GetString(annotations, path.Join(keyPrefix, monitoringAgent))
	if name == "" {
		return nil, nil
	}
	agent := api.AgentType(name)
	var err error

	switch agent {
	case api.AgentCoreOSPrometheus:
		var prom api.PrometheusSpec

		prom.Namespace = meta.GetString(annotations, path.Join(keyPrefix, serviceMonitorNamespace))
		if prom.Namespace == "" {
			return nil, fmt.Errorf("missing %s annotation", path.Join(keyPrefix, serviceMonitorNamespace))
		}

		prom.Labels, err = meta.GetMap(annotations, path.Join(keyPrefix, serviceMonitorLabels))
		if err != nil {
			return nil, err
		}
		if len(prom.Labels) <= 0 {
			return nil, fmt.Errorf("missing %s annotation", path.Join(keyPrefix, serviceMonitorLabels))
		}

		port, err := meta.GetInt(annotations, path.Join(keyPrefix, serviceMonitorPort))
		if err != nil {
			return nil, err
		}
		if port == 0 {
			prom.Port = defaultExporterPort
		} else {
			prom.Port = int32(port)
		}

		prom.Interval = meta.GetString(annotations, path.Join(keyPrefix, serviceMonitorScrapeInterval))

		return &api.AgentSpec{Agent: agent, Prometheus: &prom}, nil
	case api.AgentPrometheusBuiltin:
		var prom api.PrometheusSpec

		port, err := meta.GetInt(annotations, path.Join(keyPrefix, serviceMonitorPort))
		if err != nil {
			return nil, err
		}
		if port == 0 {
			prom.Port = defaultExporterPort
		} else {
			prom.Port = int32(port)
		}
		return &api.AgentSpec{Agent: agent, Prometheus: &prom}, nil
	default:
		return nil, fmt.Errorf("unknown monitoring agent %s", name)
	}
}
