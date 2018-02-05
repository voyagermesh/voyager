package v1beta1

import (
	"github.com/appscode/voyager/apis/voyager"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const VoyagerFinalizer = "voyager.appscode.com"

func (r Ingress) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ResourceTypeIngress + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "voyager"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   voyager.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   ResourceNameIngress,
				Plural:     ResourceTypeIngress,
				Kind:       ResourceKindIngress,
				ShortNames: []string{"ing"},
			},
		},
	}
}

func (c Certificate) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ResourceTypeCertificate + "." + SchemeGroupVersion.Group,
			Labels: map[string]string{"app": "voyager"},
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group:   voyager.GroupName,
			Version: SchemeGroupVersion.Version,
			Scope:   apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Singular:   ResourceNameCertificate,
				Plural:     ResourceTypeCertificate,
				Kind:       ResourceKindCertificate,
				ShortNames: []string{"cert"},
			},
		},
	}
}
