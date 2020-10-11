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

package agents

import (
	"kmodules.xyz/monitoring-agent-api/agents/prometheus"
	"kmodules.xyz/monitoring-agent-api/agents/prometheusbuiltin"
	"kmodules.xyz/monitoring-agent-api/agents/prometheusoperator"
	api "kmodules.xyz/monitoring-agent-api/api/v1"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"k8s.io/client-go/kubernetes"
)

func New(at api.AgentType, k8sClient kubernetes.Interface, promClient prom.MonitoringV1Interface) api.Agent {
	switch at {
	case api.AgentPrometheus:
		return prometheus.New()
	case api.AgentPrometheusOperator:
		return prometheusoperator.New(at, k8sClient, promClient)
	case api.AgentPrometheusBuiltin:
		return prometheusbuiltin.New(k8sClient)
	}
	return nil
}
