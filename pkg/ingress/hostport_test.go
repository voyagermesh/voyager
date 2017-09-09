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

func TestHostPortController_IsExists(t *testing.T) {
	testCases := map[*hostPortController]bool{
		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeHostPort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.DaemonSet{
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
							api.LBType: api.LBTypeHostPort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.DaemonSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fake-foo",
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

		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeHostPort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.DaemonSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&apiv1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fake-foo",
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
		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeHostPort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.DaemonSet{
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
							Name:      api.VoyagerPrefix + "fake-foo",
							Namespace: "bar",
						},
					},
				),
			},
		}: false,

		{
			controller: &controller{
				Ingress: &api.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							api.LBType: api.LBTypeHostPort,
						},
					},
				},
				KubeClient: fake.NewSimpleClientset(
					&extensions.DaemonSet{
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
	}

	for k, v := range testCases {
		fmt.Println(k.Ingress.Name, k.Ingress.Annotations)
		assert.Equal(t, v, k.IsExists())
	}
}
