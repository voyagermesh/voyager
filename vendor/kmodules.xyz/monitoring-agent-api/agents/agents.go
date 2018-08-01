package agents

import (
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	ecs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"kmodules.xyz/monitoring-agent-api/agents/coreosprometheusoperator"
	"kmodules.xyz/monitoring-agent-api/agents/prometheusbuiltin"
	api "kmodules.xyz/monitoring-agent-api/api/v1"
)

func New(at api.AgentType, k8sClient kubernetes.Interface, extClient ecs.ApiextensionsV1beta1Interface, promClient prom.MonitoringV1Interface) api.Agent {
	switch at {
	case api.AgentCoreOSPrometheus, api.DeprecatedAgentCoreOSPrometheus:
		return coreosprometheusoperator.New(at, k8sClient, extClient, promClient)
	case api.AgentPrometheusBuiltin:
		return prometheusbuiltin.New(k8sClient)
	}
	return nil
}
