package dynamic

import (
	"github.com/appscode/kutil/core/v1"
	discovery_util "github.com/appscode/kutil/discovery"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func DetectWorkload(config *rest.Config, resource schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	kc := kubernetes.NewForConfigOrDie(config)
	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	obj, err := dc.Resource(resource).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return findWorkload(kc, dc, obj)
}

func findWorkload(kc kubernetes.Interface, dc dynamic.Interface, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	for _, ref := range m.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			gvk := schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind)
			gvr, err := discovery_util.ResourceForGVK(kc.Discovery(), gvk)
			if err != nil {
				return nil, err
			}
			parent, err := dc.Resource(gvr).Namespace(m.GetNamespace()).Get(ref.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return findWorkload(kc, dc, parent)
		}
	}
	return obj, nil
}

func RemoveOwnerReferenceForItems(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	items []string,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	var errs []error
	for _, name := range items {
		item, err := ri.Get(name, metav1.GetOptions{})
		if err != nil {
			if !kerr.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}
		if _, _, err := Patch(c, gvr, item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.RemoveOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func RemoveOwnerReferenceForSelector(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	selector labels.Selector,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	list, err := ri.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	var errs []error
	for _, item := range list.Items {
		if _, _, err := Patch(c, gvr, &item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.RemoveOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func EnsureOwnerReferenceForItems(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	items []string,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	var errs []error
	for _, name := range items {
		item, err := ri.Get(name, metav1.GetOptions{})
		if err != nil {
			if !kerr.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}
		if _, _, err := Patch(c, gvr, item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.EnsureOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func EnsureOwnerReferenceForSelector(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	selector labels.Selector,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}
	list, err := ri.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	var errs []error
	for _, item := range list.Items {
		if _, _, err := Patch(c, gvr, &item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.EnsureOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}
