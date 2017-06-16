package ingress

import (
	"fmt"
	"testing"
	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestShouldHandleIngress(t *testing.T) {
	dataTables := map[*aci.Ingress]map[string]bool{
		{
			ObjectMeta: apiv1.ObjectMeta{
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "voyager",
				},
			},
		}: {
			"voyager": true,
			"":        true,
			"gce":     true,
			"gke":     true,
			"aws":     true,
		},

		{
			ObjectMeta: apiv1.ObjectMeta{
				Annotations: map[string]string{
					"ingress.appscode.com/kind": "ingress.appscode.com/v1beta1",
				},
			},
		}: {
			"voyager": true,
			"":        true,
			"gce":     true,
			"gke":     true,
			"aws":     true,
		},

		{
			ObjectMeta: apiv1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}: {
			"voyager": false,
			"":        true,
			"gce":     false,
			"gke":     false,
			"aws":     false,
		},
	}

	for k, v := range dataTables {
		for ingressClass, result := range v {
			if !assert.Equal(t, shouldHandleIngress(k, ingressClass), result) {
				t.Log("Ingress", *k, "IngressClass", ingressClass, "Expected", result)
			}
		}
	}
}

func TestEnsureServiceAnnotation(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	_, err := fakeClient.Core().Services("a").Get("b")
	assert.NotNil(t, err)

	svc, err := fakeClient.Core().Services("test-namespace").Create(&apiv1.Service{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
		},
	})

	ensureServiceAnnotations(fakeClient, &aci.Ingress{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: aci.ExtendedIngressSpec{
			Backend: &aci.ExtendedIngressBackend{
				ServiceName: "test-service.test-namespace",
			},
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.Core().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, aci.EngressKey)
	fmt.Println(svc.Annotations)

	ensureServiceAnnotations(fakeClient, &aci.Ingress{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: aci.ExtendedIngressSpec{
			Backend: &aci.ExtendedIngressBackend{
				ServiceName: "test-service.test-namespace",
			},
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.Core().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, aci.EngressKey)
	fmt.Println(svc.Annotations)

	ensureServiceAnnotations(fakeClient, &aci.Ingress{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.Core().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, aci.EngressKey)
	fmt.Println(svc.Annotations)
}
