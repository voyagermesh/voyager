package v1

import (
	"errors"

	"github.com/appscode/kutil/meta"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: prom.Group, Version: prom.Version}

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	_, err := conversion.EnforcePtr(v)
	if err != nil {
		return err
	}

	switch u := v.(type) {
	case *prom.Prometheus:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = prom.PrometheusesKind
		return nil
	case *prom.ServiceMonitor:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = prom.ServiceMonitorsKind
		return nil
	case *prom.Alertmanager:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = prom.AlertmanagersKind
		return nil
	}
	return errors.New("unknown api object type")
}

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&prom.Prometheus{},
		&prom.PrometheusList{},

		&prom.ServiceMonitor{},
		&prom.ServiceMonitorList{},

		&prom.Alertmanager{},
		&prom.AlertmanagerList{},
	)

	s.AddKnownTypes(SchemeGroupVersion,
		&metav1.Status{},
	)
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

func AddToScheme(scheme *runtime.Scheme) {
	var builder runtime.SchemeBuilder
	builder.Register(addKnownTypes)
	builder.AddToScheme(scheme)
}
