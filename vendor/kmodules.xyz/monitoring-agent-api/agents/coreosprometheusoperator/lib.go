package coreosprometheusoperator

import (
	"errors"
	"reflect"

	"github.com/appscode/go/types"
	promapi "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	prom "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	ecs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	api "kmodules.xyz/monitoring-agent-api/api/v1"
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
	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set{
			api.KeyService: sp.ServiceName() + "." + sp.GetNamespace(),
		}.String(),
	})

	vt := kutil.VerbUnchanged
		for _, item := range old.Items {
		if item != nil && (new == nil || item.Namespace != new.Prometheus.Namespace) {
			err := agent.promClient.ServiceMonitors(item.Namespace).Delete(sp.ServiceMonitorName(), nil)
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
	if new.Prometheus.Labels == nil {
		new.Prometheus.Labels = map[string]string{}
	}
	new.Prometheus.Labels[api.KeyService] = sp.ServiceName() + "." + sp.GetNamespace()

	actual, err := agent.promClient.ServiceMonitors(new.Prometheus.Namespace).Get(sp.ServiceMonitorName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return agent.createServiceMonitor(sp, new)
	} else if err != nil {
		return vt, err
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
			return vt, err
		}

		actual.Labels = new.Prometheus.Labels
		actual.ObjectMeta = agent.ensureOwnerReference(actual.ObjectMeta, *svc)
		actual.Spec.Selector = metav1.LabelSelector{
			MatchLabels: svc.Labels,
		}
		actual.Spec.NamespaceSelector = promapi.NamespaceSelector{
			MatchNames: []string{sp.GetNamespace()},
		}
		for i := range actual.Spec.Endpoints {
			actual.Spec.Endpoints[i].Interval = new.Prometheus.Interval
		}
		_, err = agent.promClient.ServiceMonitors(new.Prometheus.Namespace).Update(actual)
		return kutil.VerbUpdated, err
	}

	return vt, nil
}

func (agent *PrometheusCoreosOperator) createServiceMonitor(sp api.StatsAccessor, spec *api.AgentSpec) (kutil.VerbType, error) {
	svc, err := agent.k8sClient.CoreV1().Services(sp.GetNamespace()).Get(sp.ServiceName(), metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	var portName string
	for _, p := range svc.Spec.Ports {
		if p.Port == spec.Prometheus.Port {
			portName = p.Name
		}
	}
	if portName == "" {
		return kutil.VerbUnchanged, errors.New("no port found in stats service")
	}

	sm := &promapi.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sp.ServiceMonitorName(),
			Namespace: spec.Prometheus.Namespace,
			Labels:    spec.Prometheus.Labels,
		},
		Spec: promapi.ServiceMonitorSpec{
			NamespaceSelector: promapi.NamespaceSelector{
				MatchNames: []string{sp.GetNamespace()},
			},
			Endpoints: []promapi.Endpoint{
				{
					Port:        portName,
					Interval:    spec.Prometheus.Interval,
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
	if _, err := agent.promClient.ServiceMonitors(spec.Prometheus.Namespace).Create(sm); err != nil && !kerr.IsAlreadyExists(err) {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbCreated, nil
}

func (agent *PrometheusCoreosOperator) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	if !agent.supportsCoreOSOperator() {
		return kutil.VerbUnchanged, errors.New("cluster does not support CoreOS Prometheus operator")
	}

	old, err := agent.promClient.ServiceMonitors(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set{
			api.KeyService: sp.GetNamespace() + "." + sp.ServiceName(),
		}.String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}

	vt := kutil.VerbUnchanged
	for _, item := range old.Items {
		err := agent.promClient.ServiceMonitors(item.Namespace).Delete(sp.ServiceMonitorName(), nil)
		if err != nil && !kerr.IsNotFound(err) {
			return kutil.VerbUnchanged, err
		} else if err == nil {
			vt = kutil.VerbDeleted
		}
	}
	return vt, nil
}

func (agent *PrometheusCoreosOperator) supportsCoreOSOperator() bool {
	_, err := agent.extClient.CustomResourceDefinitions().Get(promapi.PrometheusName+"."+promapi.SchemeGroupVersion.Group, metav1.GetOptions{})
	if err != nil {
		return false
	}
	_, err = agent.extClient.CustomResourceDefinitions().Get(promapi.ServiceMonitorName+"."+promapi.SchemeGroupVersion.Group, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
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
