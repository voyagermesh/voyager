package v1beta1

import (
	core "k8s.io/api/core/v1"
)

func (r Ingress) ObjectReference() *core.ObjectReference {
	if r.APISchema() == APISchemaIngress {
		return &core.ObjectReference{
			APIVersion:      APISchemaIngress,
			Kind:            ResourceKindIngress,
			Namespace:       r.Namespace,
			Name:            r.Name,
			UID:             r.UID,
			ResourceVersion: r.ResourceVersion,
		}
	}
	return &core.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindIngress,
		Namespace:       r.Namespace,
		Name:            r.Name,
		UID:             r.UID,
		ResourceVersion: r.ResourceVersion,
	}
}

func (c Certificate) ObjectReference() *core.ObjectReference {
	return &core.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindCertificate,
		Namespace:       c.Namespace,
		Name:            c.Name,
		UID:             c.UID,
		ResourceVersion: c.ResourceVersion,
	}
}
