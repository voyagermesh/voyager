package v1beta1

import (
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func (r Ingress) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Version:       SchemeGroupVersion.Version,
		Plural:        ResourcePluralIngress,
		Singular:      ResourceSingularIngress,
		Kind:          ResourceKindIngress,
		ListKind:      ResourceKindIngress + "List",
		ShortNames:    []string{"ing"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "voyager"},
		},
		SpecDefinitionName:    "github.com/appscode/voyager/apis/voyager/v1beta1.Ingress",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
	})
}

func (c Certificate) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Version:       SchemeGroupVersion.Version,
		Plural:        ResourcePluralCertificate,
		Singular:      ResourceSingularCertificate,
		Kind:          ResourceKindCertificate,
		ListKind:      ResourceKindCertificate + "List",
		ShortNames:    []string{"cert"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "voyager"},
		},
		SpecDefinitionName:    "github.com/appscode/voyager/apis/voyager/v1beta1.Certificate",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
	})
}
