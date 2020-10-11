/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parser

import (
	"fmt"
	"path"

	kutil "kmodules.xyz/client-go"
	"kmodules.xyz/client-go/meta"
	api "kmodules.xyz/monitoring-agent-api/api/v1"
)

const (
	monitoringAgent              = "monitoring-agent"                         // Name of monitoring agent
	serviceMonitorLabels         = "service-monitor-labels"                   // map[string]string used to select Prometheus instance
	serviceMonitorPort           = "service-monitor-endpoint-port"            // Port on stats service used to expose metrics
	serviceMonitorScrapeInterval = "service-monitor-endpoint-scrape-interval" // scrape interval
)

func Parse(annotations map[string]string, keyPrefix string, defaultExporterPort int32) (*api.AgentSpec, error) {
	name, err := meta.GetStringValue(annotations, path.Join(keyPrefix, monitoringAgent))
	if err != nil && err == kutil.ErrNotFound {
		return nil, nil
	}
	agent := api.AgentType(name)

	switch agent {
	case api.AgentPrometheusOperator:
		prom := api.PrometheusSpec{
			Exporter:       api.PrometheusExporterSpec{},
			ServiceMonitor: &api.ServiceMonitorSpec{},
		}

		prom.ServiceMonitor.Labels, err = meta.GetMapValue(annotations, path.Join(keyPrefix, serviceMonitorLabels))
		if err != nil && err != kutil.ErrNotFound {
			return nil, err
		}
		if len(prom.ServiceMonitor.Labels) <= 0 {
			return nil, fmt.Errorf("missing %s annotation", path.Join(keyPrefix, serviceMonitorLabels))
		}

		port, err := meta.GetIntValue(annotations, path.Join(keyPrefix, serviceMonitorPort))
		if err != nil && err != kutil.ErrNotFound {
			return nil, err
		}
		if port == 0 {
			prom.Exporter.Port = defaultExporterPort
		} else {
			prom.Exporter.Port = int32(port)
		}

		prom.ServiceMonitor.Interval, _ = meta.GetStringValue(annotations, path.Join(keyPrefix, serviceMonitorScrapeInterval))

		return &api.AgentSpec{Agent: agent, Prometheus: &prom}, nil
	case api.AgentPrometheusBuiltin:
		prom := api.PrometheusSpec{
			Exporter: api.PrometheusExporterSpec{},
		}

		port, err := meta.GetIntValue(annotations, path.Join(keyPrefix, serviceMonitorPort))
		if err != nil && err != kutil.ErrNotFound {
			return nil, err
		}
		if port == 0 {
			prom.Exporter.Port = defaultExporterPort
		} else {
			prom.Exporter.Port = int32(port)
		}
		return &api.AgentSpec{Agent: agent, Prometheus: &prom}, nil
	default:
		return nil, fmt.Errorf("unknown monitoring agent %s", name)
	}
}
