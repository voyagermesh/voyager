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

package prometheusbuiltin

import (
	"context"
	"fmt"

	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	api "kmodules.xyz/monitoring-agent-api/api/v1"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	svc, e2 := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(context.TODO(), sp.ServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(e2) {
		return kutil.VerbUnchanged, e2
	}
	_, vt, err := core_util.PatchService(context.TODO(), agent.k8sClient, svc, func(in *core.Service) *core.Service {
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
		if new.Prometheus.Exporter.Port > 0 {
			in.Annotations["prometheus.io/port"] = fmt.Sprintf("%d", new.Prometheus.Exporter.Port)
		} else {
			delete(in.Annotations, "prometheus.io/port")
		}
		return in
	}, metav1.PatchOptions{})
	return vt, err
}

func (agent *PrometheusBuiltin) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	svc, e2 := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(context.TODO(), sp.ServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(e2) {
		return kutil.VerbUnchanged, e2
	}
	_, vt, err := core_util.PatchService(context.TODO(), agent.k8sClient, svc, func(in *core.Service) *core.Service {
		if in.Annotations != nil {
			delete(in.Annotations, "prometheus.io/scrape")
			delete(in.Annotations, "prometheus.io/scheme")
			delete(in.Annotations, "prometheus.io/path")
			delete(in.Annotations, "prometheus.io/port")
		}
		return in
	}, metav1.PatchOptions{})
	return vt, err
}
