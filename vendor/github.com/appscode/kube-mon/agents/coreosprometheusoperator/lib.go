package coreosprometheusoperator

import (
	"errors"
	"reflect"

	"github.com/appscode/kube-mon/api"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	ecs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const serviceKey = "monitoring.appscode.com/service-key"

// PrometheusCoreosOperator creates `ServiceMonitor` so that CoreOS Prometheus operator can generate necessary config for Prometheus.
type PrometheusCoreosOperator struct {
	k8sClient  kubernetes.Interface
	promClient prom.MonitoringV1Interface
	extClient  ecs.ApiextensionsV1beta1Interface
}

func New(k8sClient kubernetes.Interface, extClient ecs.ApiextensionsV1beta1Interface, promClient prom.MonitoringV1Interface) api.Agent {
	return &PrometheusCoreosOperator{
		k8sClient:  k8sClient,
		extClient:  extClient,
		promClient: promClient,
	}
}

func (agent *PrometheusCoreosOperator) CreateOrUpdate(sp api.StatsAccessor, new *api.AgentSpec) (bool, error) {
	if !agent.supportsCoreOSOperator() {
		return false, errors.New("cluster does not support CoreOS Prometheus operator")
	}
	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set{
			serviceKey: sp.GetNamespace() + "." + sp.ServiceName(),
		}.String(),
	})

	ok := false
	oldItems := old.(*prom.ServiceMonitorList)
	for _, item := range oldItems.Items {
		if item != nil && (new == nil || item.Namespace != new.Prometheus.Namespace) {
			err := agent.promClient.ServiceMonitors(item.Namespace).Delete(sp.ServiceMonitorName(), nil)
			if err != nil && !kerr.IsNotFound(err) {
				return false, err
			} else if err == nil {
				ok = true
			}
		}
	}
	if new == nil {
		return ok, nil
	}

	// Unique Label Selector for ServiceMonitor
	if new.Prometheus.Labels == nil {
		new.Prometheus.Labels = map[string]string{}
	}
	new.Prometheus.Labels[serviceKey] = sp.GetNamespace() + "." + sp.ServiceName()

	actual, err := agent.promClient.ServiceMonitors(new.Prometheus.Namespace).Get(sp.ServiceMonitorName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return agent.createServiceMonitor(sp, new)
	} else if err != nil {
		return ok, err
	}

	update := false
	if !reflect.DeepEqual(actual.Labels, new.Prometheus.Labels) {
		update = true
	}

	if !update {
		for _, e := range actual.Spec.Endpoints {
			if e.Interval != new.Prometheus.Interval {
				update = true
				break
			}
		}
	}

	if update {
		svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(sp.ServiceName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		actual.Labels = new.Prometheus.Labels
		actual.Spec.Selector = metav1.LabelSelector{
			MatchLabels: svc.Labels,
		}
		actual.Spec.NamespaceSelector = prom.NamespaceSelector{
			MatchNames: []string{sp.GetNamespace()},
		}
		for i := range actual.Spec.Endpoints {
			actual.Spec.Endpoints[i].Interval = new.Prometheus.Interval
		}
		_, err = agent.promClient.ServiceMonitors(new.Prometheus.Namespace).Update(actual)
		return update, err
	}

	return update, nil
}

func (agent *PrometheusCoreosOperator) createServiceMonitor(sp api.StatsAccessor, spec *api.AgentSpec) (bool, error) {
	svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(sp.ServiceName(), metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	var portName string
	for _, p := range svc.Spec.Ports {
		if p.Port == spec.Prometheus.Port {
			portName = p.Name
		}
	}
	if portName == "" {
		return false, errors.New("no port found in stats service")
	}

	sm := &prom.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sp.ServiceMonitorName(),
			Namespace: spec.Prometheus.Namespace,
			Labels:    spec.Prometheus.Labels,
		},
		Spec: prom.ServiceMonitorSpec{
			NamespaceSelector: prom.NamespaceSelector{
				MatchNames: []string{sp.GetNamespace()},
			},
			Endpoints: []prom.Endpoint{
				{
					Port:     portName,
					Interval: spec.Prometheus.Interval,
					Path:     sp.Path(),
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: svc.Labels,
			},
		},
	}
	if _, err := agent.promClient.ServiceMonitors(spec.Prometheus.Namespace).Create(sm); err != nil && !kerr.IsAlreadyExists(err) {
		return false, err
	}
	return true, nil
}

func (agent *PrometheusCoreosOperator) Delete(sp api.StatsAccessor) (bool, error) {
	if !agent.supportsCoreOSOperator() {
		return false, errors.New("cluster does not support CoreOS Prometheus operator")
	}

	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set{
			serviceKey: sp.GetNamespace() + "." + sp.ServiceName(),
		}.String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		return false, err
	}

	ok := false
	oldItems := old.(*prom.ServiceMonitorList)
	for _, item := range oldItems.Items {
		err := agent.promClient.ServiceMonitors(item.Namespace).Delete(sp.ServiceMonitorName(), nil)
		if err != nil && !kerr.IsNotFound(err) {
			return false, err
		} else if err == nil {
			ok = true
		}
	}
	return ok, nil
}

func (agent *PrometheusCoreosOperator) supportsCoreOSOperator() bool {
	_, err := agent.extClient.CustomResourceDefinitions().Get(prom.PrometheusName+"."+prom.Group, metav1.GetOptions{})
	if err != nil {
		return false
	}
	_, err = agent.extClient.CustomResourceDefinitions().Get(prom.ServiceMonitorName+"."+prom.Group, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}
