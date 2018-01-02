package util

import (
	"errors"

	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return api.SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	_, err := conversion.EnforcePtr(v)
	if err != nil {
		return err
	}

	switch u := v.(type) {
	case *api.Ingress:
		u.APIVersion = api.SchemeGroupVersion.String()
		u.Kind = api.ResourceKindIngress
		return nil
	case *api.Certificate:
		u.APIVersion = api.SchemeGroupVersion.String()
		u.Kind = api.ResourceKindCertificate
		return nil
	}
	return errors.New("unknown api object type")
}
