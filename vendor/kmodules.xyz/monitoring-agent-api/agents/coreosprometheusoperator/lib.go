/*
Copyright The Kmodules Authors.

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

package coreosprometheusoperator

import (
	"context"
	"errors"
	"reflect"

	kutil "kmodules.xyz/client-go"
	api "kmodules.xyz/monitoring-agent-api/api/v1"

	"github.com/appscode/go/types"
	promapi "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	prom "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	ecs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// PrometheusCoreosOperator creates `ServiceMonitor` so that CoreOS Prometheus operator can generate necessary config for Prometheus.
type PrometheusCoreosOperator struct {
	at         api.AgentType
	k8sClient  kubernetes.Interface
	promClient prom.MonitoringV1Interface
	extClient  ecs.ApiextensionsV1beta1Interface
}

func New(at api.AgentType, k8sClient kubernetes.Interface, extClient ecs.ApiextensionsV1beta1Interface, promClient prom.MonitoringV1Interface) api.Agent {
	return &PrometheusCoreosOperator{
		at:         at,
		k8sClient:  k8sClient,
		extClient:  extClient,
		promClient: promClient,
	}
}

func (agent *PrometheusCoreosOperator) GetType() api.AgentType {
	return agent.at
}

func (agent *PrometheusCoreosOperator) CreateOrUpdate(sp api.StatsAccessor, new *api.AgentSpec) (kutil.VerbType, error) {
	if !agent.supportsCoreOSOperator() {
		return kutil.VerbUnchanged, errors.New("cluster does not support CoreOS Prometheus operator")
	}
	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set{
			api.KeyService: sp.ServiceName() + "." + sp.GetNamespace(),
		}.String(),
	})
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	vt := kutil.VerbUnchanged
	for _, item := range old.Items {
		if item != nil && (new == nil || item.Namespace != new.Prometheus.ServiceMonitor.Namespace) {
			err := agent.promClient.ServiceMonitors(item.Namespace).Delete(context.TODO(), sp.ServiceMonitorName(), metav1.DeleteOptions{})
			if err != nil && !kerr.IsNotFound(err) {
				return kutil.VerbUnchanged, err
			} else if err == nil {
				vt = kutil.VerbDeleted
			}
		}
	}
	if new == nil {
		return vt, nil
	}

	// Unique Label Selector for ServiceMonitor
	if new.Prometheus.ServiceMonitor.Labels == nil {
		new.Prometheus.ServiceMonitor.Labels = map[string]string{}
	}
	new.Prometheus.ServiceMonitor.Labels[api.KeyService] = sp.ServiceName() + "." + sp.GetNamespace()

	actual, err := agent.promClient.ServiceMonitors(new.Prometheus.ServiceMonitor.Namespace).Get(context.TODO(), sp.ServiceMonitorName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return agent.createServiceMonitor(sp, new)
	} else if err != nil {
		return vt, err
	}

	update := false
	if !reflect.DeepEqual(actual.Labels, new.Prometheus.ServiceMonitor.Labels) {
		update = true
	}

	if !update {
		for _, e := range actual.Spec.Endpoints {
			if e.Interval != new.Prometheus.ServiceMonitor.Interval {
				update = true
				break
			}
		}
	}

	if update {
		svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(context.TODO(), sp.ServiceName(), metav1.GetOptions{})
		if err != nil {
			return vt, err
		}

		actual.Labels = new.Prometheus.ServiceMonitor.Labels
		actual.ObjectMeta = agent.ensureOwnerReference(actual.ObjectMeta, *svc)
		actual.Spec.Selector = metav1.LabelSelector{
			MatchLabels: svc.Labels,
		}
		actual.Spec.NamespaceSelector = promapi.NamespaceSelector{
			MatchNames: []string{sp.GetNamespace()},
		}
		for i := range actual.Spec.Endpoints {
			actual.Spec.Endpoints[i].Interval = new.Prometheus.ServiceMonitor.Interval
		}
		_, err = agent.promClient.ServiceMonitors(new.Prometheus.ServiceMonitor.Namespace).Update(context.TODO(), actual, metav1.UpdateOptions{})
		return kutil.VerbUpdated, err
	}

	return vt, nil
}

func (agent *PrometheusCoreosOperator) createServiceMonitor(sp api.StatsAccessor, spec *api.AgentSpec) (kutil.VerbType, error) {
	svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(context.TODO(), sp.ServiceName(), metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	var portName string
	for _, p := range svc.Spec.Ports {
		if p.Port == spec.Prometheus.Exporter.Port {
			portName = p.Name
		}
	}
	if portName == "" {
		return kutil.VerbUnchanged, errors.New("no port found in stats service")
	}

	sm := &promapi.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sp.ServiceMonitorName(),
			Namespace: spec.Prometheus.ServiceMonitor.Namespace,
			Labels:    spec.Prometheus.ServiceMonitor.Labels,
		},
		Spec: promapi.ServiceMonitorSpec{
			NamespaceSelector: promapi.NamespaceSelector{
				MatchNames: []string{sp.GetNamespace()},
			},
			Endpoints: []promapi.Endpoint{
				{
					Port:        portName,
					Interval:    spec.Prometheus.ServiceMonitor.Interval,
					Path:        sp.Path(),
					HonorLabels: true,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: svc.Labels,
			},
		},
	}
	sm.ObjectMeta = agent.ensureOwnerReference(sm.ObjectMeta, *svc)
	if _, err := agent.promClient.ServiceMonitors(spec.Prometheus.ServiceMonitor.Namespace).Create(context.TODO(), sm, metav1.CreateOptions{}); err != nil && !kerr.IsAlreadyExists(err) {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbCreated, nil
}

func (agent *PrometheusCoreosOperator) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	if !agent.supportsCoreOSOperator() {
		return kutil.VerbUnchanged, errors.New("cluster does not support CoreOS Prometheus operator")
	}

	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set{
			api.KeyService: sp.GetNamespace() + "." + sp.ServiceName(),
		}.String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}

	vt := kutil.VerbUnchanged
	for _, item := range old.Items {
		err := agent.promClient.ServiceMonitors(item.Namespace).Delete(context.TODO(), sp.ServiceMonitorName(), metav1.DeleteOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return kutil.VerbUnchanged, err
		} else if err == nil {
			vt = kutil.VerbDeleted
		}
	}
	return vt, nil
}

func (agent *PrometheusCoreosOperator) supportsCoreOSOperator() bool {
	_, err := agent.extClient.CustomResourceDefinitions().Get(context.TODO(), promapi.PrometheusName+"."+promapi.SchemeGroupVersion.Group, metav1.GetOptions{})
	if err != nil {
		return false
	}
	_, err = agent.extClient.CustomResourceDefinitions().Get(context.TODO(), promapi.ServiceMonitorName+"."+promapi.SchemeGroupVersion.Group, metav1.GetOptions{})
	return err == nil
}

func (agent *PrometheusCoreosOperator) ensureOwnerReference(in metav1.ObjectMeta, svc corev1.Service) metav1.ObjectMeta {
	fi := -1
	for i, ref := range in.OwnerReferences {
		if ref.Kind == "Service" && ref.Name == svc.Name {
			fi = i
			break
		}
	}
	if fi == -1 {
		in.OwnerReferences = append(in.OwnerReferences, metav1.OwnerReference{})
		fi = len(in.OwnerReferences) - 1
	}
	in.OwnerReferences[fi].APIVersion = "v1"
	in.OwnerReferences[fi].Kind = "Service"
	in.OwnerReferences[fi].Name = svc.Name
	in.OwnerReferences[fi].UID = svc.UID
	in.OwnerReferences[fi].BlockOwnerDeletion = types.TrueP()
	return in
}
