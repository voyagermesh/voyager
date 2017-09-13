package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (r Ingress) ObjectReference() runtime.Object {
	if r.APISchema() == APISchemaIngress {
		return &apiv1.ObjectReference{
			APIVersion:      APISchemaIngress,
			Kind:            ResourceKindIngress,
			Namespace:       r.Namespace,
			Name:            r.Name,
			UID:             r.UID,
			ResourceVersion: r.ResourceVersion,
		}
	}
	return &apiv1.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindIngress,
		Namespace:       r.Namespace,
		Name:            r.Name,
		UID:             r.UID,
		ResourceVersion: r.ResourceVersion,
	}
}

func (c Certificate) ObjectReference() runtime.Object {
	return &apiv1.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindCertificate,
		Namespace:       c.Namespace,
		Name:            c.Name,
		UID:             c.UID,
		ResourceVersion: c.ResourceVersion,
	}
}
