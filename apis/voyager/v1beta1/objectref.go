package v1beta1

import (
	apiv1 "k8s.io/api/core/v1"
)

func (r Ingress) ObjectReference() *apiv1.ObjectReference {
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

func (c Certificate) ObjectReference() *apiv1.ObjectReference {
	return &apiv1.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindCertificate,
		Namespace:       c.Namespace,
		Name:            c.Name,
		UID:             c.UID,
		ResourceVersion: c.ResourceVersion,
	}
}
