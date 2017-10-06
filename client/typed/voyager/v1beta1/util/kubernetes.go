package util

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return api.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
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
