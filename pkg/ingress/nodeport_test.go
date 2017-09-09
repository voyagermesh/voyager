package ingress

import (
	"fmt"
	"testing"

	api "github.com/appscode/voyager/apis/voyager"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestNodePortController_IsExists(t *testing.T) {
	testCases := map[*nodePortController]bool{
		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeNodePort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.Deployment{
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
							api.LBType: api.LBTypeNodePort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.Deployment{
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
		fmt.Println(k.Ingress.Name, k.Ingress.Annotations)
		assert.Equal(t, v, k.IsExists())
	}
}
