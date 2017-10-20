package ingress

import (
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLoadBalancerController_IsExists(t *testing.T) {
	testCases := map[*loadBalancerController]bool{
		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeLoadBalancer,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&apiv1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&apiv1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},
				),
			},
		}: true,

		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeLoadBalancer,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fakefoo",
							Namespace: "bar",
						},
					},

					&apiv1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&apiv1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},
				),
			},
		}: false,
	}

	for k, v := range testCases {
		t.Log(k.Ingress.Name, k.Ingress.Annotations)
		assert.Equal(t, v, k.IsExists())
	}
}
