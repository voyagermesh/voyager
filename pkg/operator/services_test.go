package operator

import (
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func TestEnsureServiceAnnotation(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	_, err := fakeClient.CoreV1().Services("a").Get("b", metav1.GetOptions{})
	assert.NotNil(t, err)

	svc, err := fakeClient.CoreV1().Services("test-namespace").Create(&apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
		},
	})

	op := &Operator{KubeClient: fakeClient}

	op.ensureEgressAnnotations(&api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.IngressSpec{
			Backend: &api.HTTPIngressBackend{
				IngressBackend: api.IngressBackend{ServiceName: "test-service.test-namespace"},
			},
		},
	}, svc)

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EgressPoints)

	op.ensureEgressAnnotations(&api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.IngressSpec{
			Backend: &api.HTTPIngressBackend{
				IngressBackend: api.IngressBackend{ServiceName: "test-service.test-namespace"},
			},
		},
	}, svc)

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EgressPoints)

	op.ensureEgressAnnotations(&api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}, svc)

	svc, err = fakeClient.CoreV1().Services("test-namespace").Get("test-service", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, svc.Annotations)
	assert.Contains(t, svc.Annotations, api.EgressPoints)
}
