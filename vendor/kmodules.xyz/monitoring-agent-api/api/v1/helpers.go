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

package v1

import (
	core "k8s.io/api/core/v1"
)

func (agent *AgentSpec) SetDefaults() {
	if agent == nil {
		return
	}

	if agent.Prometheus == nil {
		return
	}

	if agent.Prometheus.Exporter == nil {
		agent.Prometheus.Exporter = &PrometheusExporterSpec{}
	}

	if agent.Prometheus.Port > 0 && agent.Prometheus.Exporter.Port == 0 {
		agent.Prometheus.Exporter.Port = agent.Prometheus.Port
	}
	agent.Prometheus.Port = 0

	if len(agent.Args) > 0 && len(agent.Prometheus.Exporter.Args) == 0 {
		agent.Prometheus.Exporter.Args = agent.Args
	}
	agent.Args = nil

	if len(agent.Env) > 0 && len(agent.Prometheus.Exporter.Env) == 0 {
		agent.Prometheus.Exporter.Env = agent.Env
	}
	agent.Env = nil

	if !resourceIsZero(agent.Resources) && resourceIsZero(agent.Prometheus.Exporter.Resources) {
		agent.Prometheus.Exporter.Resources = agent.Resources
	}
	agent.Resources = core.ResourceRequirements{}

	if agent.SecurityContext != nil && agent.Prometheus.Exporter.SecurityContext == nil {
		agent.Prometheus.Exporter.SecurityContext = agent.SecurityContext
	}
	agent.SecurityContext = nil

	if agent.Agent == AgentPrometheusOperator || agent.Agent == AgentCoreOSPrometheus || agent.Agent == DeprecatedAgentCoreOSPrometheus {
		if agent.Prometheus.ServiceMonitor == nil {
			agent.Prometheus.ServiceMonitor = &ServiceMonitorSpec{}
		}

		if len(agent.Prometheus.Labels) > 0 && len(agent.Prometheus.ServiceMonitor.Labels) == 0 {
			agent.Prometheus.ServiceMonitor.Labels = agent.Prometheus.Labels
		}
		agent.Prometheus.Labels = nil

		if agent.Prometheus.Interval != "" && agent.Prometheus.ServiceMonitor.Interval == "" {
			agent.Prometheus.ServiceMonitor.Interval = agent.Prometheus.Interval
		}
		agent.Prometheus.Interval = ""
	}
}

func resourceIsZero(r core.ResourceRequirements) bool {
	return len(r.Limits) == 0 && len(r.Requests) == 0
}
