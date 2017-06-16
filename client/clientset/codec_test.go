package clientset

import (
	"fmt"
	"reflect"
	"testing"

	aci "github.com/appscode/voyager/api"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/pkg/api"
)

func TestDefaultGroupVersion(t *testing.T) {
	i := &aci.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	gv, err := schema.ParseGroupVersion("appscode.com/v1beta1")
	if err != nil {
		fmt.Println(err)
	}
	// if appscode.com/v1beta1 is not enabled, return an error
	if !api.Registry.IsEnabledVersion(gv) {
		fmt.Println("appscode.com/v1beta1 is not enabled")
	}

	fmt.Println(*i)
}

func TestSetDefault(t *testing.T) {
	metadata := &metav1.TypeMeta{
		Kind:       "Ingress",
		APIVersion: "appscode.com/v1beta1",
	}
	var obj runtime.Object

	obj, err := setDefaultType(metadata)
	fmt.Println(obj, err)
	assert.NotNil(t, obj)
	fmt.Println(reflect.ValueOf(obj).Type().String())
}
