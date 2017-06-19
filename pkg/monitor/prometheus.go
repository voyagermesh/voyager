package monitor

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/voyager/api"
	"github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	_ "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type PrometheusController struct {
	kubeClient clientset.Interface
	promClient v1alpha1.MonitoringV1alpha1Interface
}

func NewPrometheusController(kubeClient clientset.Interface, promClient v1alpha1.MonitoringV1alpha1Interface) Monitor {
	return &PrometheusController{
		kubeClient: kubeClient,
		promClient: promClient,
	}
}

func (c *PrometheusController) AddMonitor(r *api.Ingress, spec *api.MonitorSpec) error {
	if !c.SupportsCoreOSOperator() {
		return errors.New("Cluster does not support CoreOS Prometheus operator")
	}
	return c.ensureServiceMonitor(r, spec, spec)
}

func (c *PrometheusController) UpdateMonitor(r *api.Ingress, old, new *api.MonitorSpec) error {
	if !c.SupportsCoreOSOperator() {
		return errors.New("Cluster does not support CoreOS Prometheus operator")
	}
	return c.ensureServiceMonitor(r, old, new)
}

func (c *PrometheusController) DeleteMonitor(r *api.Ingress, spec *api.MonitorSpec) error {
	if !c.SupportsCoreOSOperator() {
		return errors.New("Cluster does not support CoreOS Prometheus operator")
	}
	if err := c.promClient.ServiceMonitors(spec.Prometheus.Namespace).Delete(getServiceMonitorName(r), nil); !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *PrometheusController) SupportsCoreOSOperator() bool {
	_, err := c.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get("prometheus."+prom.TPRGroup, metav1.GetOptions{})
	if err != nil {
		return false
	}
	_, err = c.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get("service-monitor."+prom.TPRGroup, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func (c *PrometheusController) ensureServiceMonitor(r *api.Ingress, old, new *api.MonitorSpec) error {
	name := getServiceMonitorName(r)
	if old != nil && (new == nil || old.Prometheus.Namespace != new.Prometheus.Namespace) {
		err := c.promClient.ServiceMonitors(old.Prometheus.Namespace).Delete(name, nil)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
		if new == nil {
			return nil
		}
	}

	actual, err := c.promClient.ServiceMonitors(new.Prometheus.Namespace).Get(name)
	if kerr.IsNotFound(err) {
		return c.createServiceMonitor(r, new)
	} else if err != nil {
		return err
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
		actual.Labels = new.Prometheus.Labels
		for i := range actual.Spec.Endpoints {
			actual.Spec.Endpoints[i].Interval = new.Prometheus.Interval
		}
		_, err := c.promClient.ServiceMonitors(new.Prometheus.Namespace).Update(actual)
		return err
	}

	return nil
}

func (c *PrometheusController) createServiceMonitor(r *api.Ingress, spec *api.MonitorSpec) error {
	sm := &prom.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getServiceMonitorName(r),
			Namespace: spec.Prometheus.Namespace,
			Labels:    spec.Prometheus.Labels,
		},
		Spec: prom.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: r.OffshootLabels(),
			},
			NamespaceSelector: prom.NamespaceSelector{
				MatchNames: []string{r.Namespace},
			},
			Endpoints: []prom.Endpoint{
				{
					TargetPort: spec.Prometheus.TargetPort,
					Interval:   spec.Prometheus.Interval,
					Path:       fmt.Sprintf("/%s/namespaces/%s/ingresses/%s/metrics", r.APISchema(), r.Namespace, r.Name),
				},
			},
		},
	}
	if _, err := c.promClient.ServiceMonitors(spec.Prometheus.Namespace).Create(sm); !kerr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func getServiceMonitorName(r *api.Ingress) string {
	return api.VoyagerPrefix + r.Namespace + "-" + r.Name
}
