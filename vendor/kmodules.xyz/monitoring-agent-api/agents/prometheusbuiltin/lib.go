package prometheusbuiltin

import (
	"fmt"

	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kmodules.xyz/monitoring-agent-api/api"
)

// PrometheusBuiltin applies `prometheus.io` annotations on stats service so that Prometheus can scrape this stats service.
// ref: https://github.com/prometheus/prometheus/blob/a51c500e30e96df4026282c8a4f743cf6a8827f1/documentation/examples/prometheus-kubernetes.yml#L136
type PrometheusBuiltin struct {
	k8sClient kubernetes.Interface
}

func New(k8sClient kubernetes.Interface) api.Agent {
	return &PrometheusBuiltin{k8sClient: k8sClient}
}

func (agent *PrometheusBuiltin) GetType() api.AgentType {
	return api.AgentPrometheusBuiltin
}

func (agent *PrometheusBuiltin) CreateOrUpdate(sp api.StatsAccessor, new *api.AgentSpec) (kutil.VerbType, error) {
	svc, e2 := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(sp.ServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(e2) {
		return kutil.VerbUnchanged, e2
	}
	_, vt, err := core_util.PatchService(agent.k8sClient, svc, func(in *core.Service) *core.Service {
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations["prometheus.io/scrape"] = "true"
		if sp.Scheme() != "" {
			in.Annotations["prometheus.io/scheme"] = sp.Scheme()
		} else {
			delete(in.Annotations, "prometheus.io/scheme")
		}
		in.Annotations["prometheus.io/path"] = sp.Path()
		if new.Prometheus.Port > 0 {
			in.Annotations["prometheus.io/port"] = fmt.Sprintf("%d", new.Prometheus.Port)
		} else {
			delete(in.Annotations, "prometheus.io/port")
		}
		return in
	})
	return vt, err
}

func (agent *PrometheusBuiltin) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	svc, e2 := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(sp.ServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(e2) {
		return kutil.VerbUnchanged, e2
	}
	_, vt, err := core_util.PatchService(agent.k8sClient, svc, func(in *core.Service) *core.Service {
		if in.Annotations != nil {
			delete(in.Annotations, "prometheus.io/scrape")
			delete(in.Annotations, "prometheus.io/scheme")
			delete(in.Annotations, "prometheus.io/path")
			delete(in.Annotations, "prometheus.io/port")
		}
		return in
	})
	return vt, err
}
