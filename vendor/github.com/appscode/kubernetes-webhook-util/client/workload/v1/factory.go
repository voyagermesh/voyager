package v1

import (
	"fmt"

	"github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	ocapps "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

func NewWorkload(t metav1.TypeMeta, o metav1.ObjectMeta, tpl core.PodTemplateSpec) *v1.Workload {
	return &v1.Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec: v1.WorkloadSpec{
			Template: tpl,
		},
	}
}

func NewObjectForGVK(gvk schema.GroupVersionKind, name, ns string) (runtime.Object, error) {
	obj, err := legacyscheme.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	out, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	out.SetName(name)
	out.SetNamespace(ns)
	return obj, nil
}

func NewObjectForKind(kind v1.WorkloadKind, name, ns string) (runtime.Object, error) {
	switch kind {
	case v1.KindPod:
		return &core.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindReplicationController:
		return &core.ReplicationController{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindDeployment:
		return &appsv1beta1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindDaemonSet:
		return &extensions.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindReplicaSet:
		return &extensions.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindStatefulSet:
		return &appsv1beta1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindJob:
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindCronJob:
		return &batchv1beta1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	case v1.KindDeploymentConfig:
		return &ocapps.DeploymentConfig{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}, nil
	default:
		return nil, fmt.Errorf("unknown kind %s", kind)
	}
}

func newWithObject(t metav1.TypeMeta, o metav1.ObjectMeta, replicas *int32, sel *metav1.LabelSelector, tpl core.PodTemplateSpec, obj runtime.Object) *v1.Workload {
	return &v1.Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec: v1.WorkloadSpec{
			Replicas: replicas,
			Selector: sel,
			Template: tpl,
		},
		Object: obj,
	}
}

// ref: https://github.com/kubernetes/kubernetes/blob/4f083dee54539b0ca24ddc55d53921f5c2efc0b9/pkg/kubectl/cmd/util/factory_client_access.go#L221
func ConvertToWorkload(obj runtime.Object) (*v1.Workload, error) {
	switch t := obj.(type) {
	case *core.Pod:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, nil, core.PodTemplateSpec{ObjectMeta: t.ObjectMeta, Spec: t.Spec}, obj), nil
		// ReplicationController
	case *core.ReplicationController:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, &metav1.LabelSelector{MatchLabels: t.Spec.Selector}, *t.Spec.Template, obj), nil
		// Deployment
	case *extensions.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
		// DaemonSet
	case *extensions.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, t.Spec.Selector, t.Spec.Template, obj), nil
		// ReplicaSet
	case *extensions.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Replicas, t.Spec.Selector, t.Spec.Template, obj), nil
		// Job
	case *batchv1.Job:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, t.Spec.Selector, t.Spec.Template, obj), nil
		// CronJob
	case *batchv1beta1.CronJob:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, t.Spec.JobTemplate.Spec.Selector, t.Spec.JobTemplate.Spec.Template, obj), nil
		// DeploymentConfig
	case *ocapps.DeploymentConfig:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		var replicas *int32
		if t.Spec.Replicas > 0 {
			replicas = &t.Spec.Replicas
		}
		return newWithObject(t.TypeMeta, t.ObjectMeta, replicas, &metav1.LabelSelector{MatchLabels: t.Spec.Selector}, *t.Spec.Template, obj), nil
	default:
		return nil, fmt.Errorf("the object is not a pod or does not have a pod template")
	}
}

func ApplyWorkload(obj runtime.Object, w *v1.Workload) error {
	switch t := obj.(type) {
	case *core.Pod:
		t.ObjectMeta = w.ObjectMeta
		t.Spec = w.Spec.Template.Spec
		// ReplicationController
	case *core.ReplicationController:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = &w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
		// Deployment
	case *extensions.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1beta1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1beta2.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
		// DaemonSet
	case *extensions.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta2.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// ReplicaSet
	case *extensions.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1beta2.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1beta2.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
	case *appsv1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = w.Spec.Replicas
		}
		// Job
	case *batchv1.Job:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// CronJob
	case *batchv1beta1.CronJob:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.JobTemplate.Spec.Template = w.Spec.Template
		// DeploymentConfig
	case *ocapps.DeploymentConfig:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = &w.Spec.Template
		if w.Spec.Replicas != nil {
			t.Spec.Replicas = *w.Spec.Replicas
		}
	default:
		return fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return nil
}
