package api

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "appscode.com"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to apiv1.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Ingress{},
		&IngressList{},

		&Certificate{},
		&CertificateList{},
	)
	return nil
}

func (r *Ingress) GetObjectKind() schema.ObjectKind       { return &r.TypeMeta }
func (obj *IngressList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }

func (obj *Certificate) GetObjectKind() schema.ObjectKind     { return &obj.TypeMeta }
func (obj *CertificateList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }
