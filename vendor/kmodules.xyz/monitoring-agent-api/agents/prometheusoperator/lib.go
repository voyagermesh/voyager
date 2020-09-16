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

package prometheusoperator

import (
	"context"

	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	api "kmodules.xyz/monitoring-agent-api/api/v1"
	prom_util "kmodules.xyz/monitoring-agent-api/prometheus/v1"

	"github.com/pkg/errors"
	promapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PrometheusOperator creates `ServiceMonitor` so that CoreOS Prometheus operator can generate necessary config for Prometheus.
type PrometheusOperator struct {
	at         api.AgentType
	k8sClient  kubernetes.Interface
	promClient prom.MonitoringV1Interface
}

func New(at api.AgentType, k8sClient kubernetes.Interface, promClient prom.MonitoringV1Interface) api.Agent {
	return &PrometheusOperator{
		at:         at,
		k8sClient:  k8sClient,
		promClient: promClient,
	}
}

func (agent *PrometheusOperator) GetType() api.AgentType {
	return agent.at
}

func (agent *PrometheusOperator) CreateOrUpdate(sp api.StatsAccessor, new *api.AgentSpec) (kutil.VerbType, error) {
	if !discovery.ExistsGroupKind(agent.k8sClient.Discovery(), promapi.SchemeGroupVersion.Group, promapi.ServiceMonitorsKind) {
		return kutil.VerbUnchanged, errors.New("cluster does not support CoreOS Prometheus operator")
	}

	svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(context.TODO(), sp.ServiceName(), metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	var portName string
	for _, p := range svc.Spec.Ports {
		if p.Port == new.Prometheus.Exporter.Port {
			portName = p.Name
		}
	}
	if portName == "" {
		return kutil.VerbUnchanged, errors.New("no port found in stats service")
	}

	smMeta := metav1.ObjectMeta{
		Name:      sp.ServiceMonitorName(),
		Namespace: sp.GetNamespace(),
	}
	owner := metav1.NewControllerRef(svc, corev1.SchemeGroupVersion.WithKind("Service"))

	_, vt, err := prom_util.CreateOrPatchServiceMonitor(context.TODO(), agent.promClient, smMeta, func(in *promapi.ServiceMonitor) *promapi.ServiceMonitor {
		in.Labels = core_util.UpsertMap(sp.ServiceMonitorAdditionalLabels(), new.Prometheus.ServiceMonitor.Labels)
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		in.Spec.NamespaceSelector = promapi.NamespaceSelector{
			MatchNames: []string{sp.GetNamespace()},
		}
		in.Spec.Endpoints = []promapi.Endpoint{
			{
				Port:        portName,
				Interval:    new.Prometheus.ServiceMonitor.Interval,
				Path:        sp.Path(),
				HonorLabels: true,
			},
		}
		in.Spec.Selector = metav1.LabelSelector{
			MatchLabels: svc.Labels,
		}
		return in
	}, metav1.PatchOptions{})

	return vt, err
}

func (agent *PrometheusOperator) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	if !discovery.ExistsGroupKind(agent.k8sClient.Discovery(), promapi.SchemeGroupVersion.Group, promapi.ServiceMonitorsKind) {
		return kutil.VerbUnchanged, errors.New("cluster does not support CoreOS Prometheus operator")
	}

	err := agent.promClient.ServiceMonitors(sp.GetNamespace()).Delete(context.TODO(), sp.ServiceMonitorName(), metav1.DeleteOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbDeleted, nil
}
