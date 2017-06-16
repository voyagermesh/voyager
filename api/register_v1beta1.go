package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/api/v1"
	versionedwatch "k8s.io/kubernetes/pkg/watch/versioned"
)

// SchemeGroupVersion is group version used to register these objects
var V1beta1SchemeGroupVersion = metav1.GroupVersion{Group: GroupName, Version: "v1beta1"}

var (
	V1beta1SchemeBuilder = runtime.NewSchemeBuilder(v1addKnownTypes, addConversionFuncs)
	V1betaAddToScheme    = V1beta1SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func v1addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(V1beta1SchemeGroupVersion,
		&Ingress{},
		&IngressList{},

		&Certificate{},
		&CertificateList{},

		&v1.ListOptions{},
	)
	versionedwatch.AddToGroupVersion(scheme, V1beta1SchemeGroupVersion)
	return nil
}
