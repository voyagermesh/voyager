package v1beta1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	voyager "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return voyager.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *voyager.Ingress:
		u.APIVersion = voyager.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *voyager.Certificate:
		u.APIVersion = voyager.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
