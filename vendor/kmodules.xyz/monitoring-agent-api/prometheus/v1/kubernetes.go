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

package v1

import (
	"errors"

	"kmodules.xyz/client-go/meta"

	api "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var SchemeGroupVersion = schema.GroupVersion{Group: api.SchemeGroupVersion.Group, Version: api.Version}

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	_, err := conversion.EnforcePtr(v)
	if err != nil {
		return err
	}

	switch u := v.(type) {
	case *api.Prometheus:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = api.PrometheusesKind
		return nil
	case *api.ServiceMonitor:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = api.ServiceMonitorsKind
		return nil
	case *api.Alertmanager:
		u.APIVersion = SchemeGroupVersion.String()
		u.Kind = api.AlertmanagersKind
		return nil
	}
	return errors.New("unknown api object type")
}

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&api.Prometheus{},
		&api.PrometheusList{},

		&api.ServiceMonitor{},
		&api.ServiceMonitorList{},

		&api.Alertmanager{},
		&api.AlertmanagerList{},
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
	utilruntime.Must(builder.AddToScheme(scheme))
}
