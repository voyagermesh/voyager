package ingress

import (
	"fmt"
	"testing"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func init() {
	testframework.Initialize()
}

func TestShouldHandleIngress(t *testing.T) {
	dataTables := map[*api.Ingress]map[string]bool{
		{
			ObjectMeta: metav1.ObjectMeta{
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"ingress.appscode.com/kind": "ingress.voyager.appscode.com/v1beta1",
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
			ObjectMeta: metav1.ObjectMeta{
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
	_, err := fakeClient.CoreV1().Services("a").Get("b")
	assert.NotNil(t, err)

	svc, err := fakeClient.CoreV1().Services("test-namespace").Create(&apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
		},
	})

	ensureServiceAnnotations(fakeClient, &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.ExtendedIngressSpec{
			Backend: &api.ExtendedIngressBackend{
				ServiceName: "test-service.test-namespace",
			},
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EngressKey)
	fmt.Println(svc.Annotations)

	ensureServiceAnnotations(fakeClient, &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.ExtendedIngressSpec{
			Backend: &api.ExtendedIngressBackend{
				ServiceName: "test-service.test-namespace",
			},
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EngressKey)
	fmt.Println(svc.Annotations)

	ensureServiceAnnotations(fakeClient, &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}, "test-namespace", "test-service")

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service")
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EngressKey)
	fmt.Println(svc.Annotations)
}
