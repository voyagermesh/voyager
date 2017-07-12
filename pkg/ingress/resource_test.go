package ingress

import (
	"testing"

	api "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func init() {
	testframework.Initialize()
}

func TestResourceIsExists(t *testing.T) {
	testCases := map[*Controller]bool{
		{
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
		}: true,

		{
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
		}: false,

		{
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
		}: false,
		{
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
		}: false,

		{
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
		}: true,

		{
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
				&apiv1.ReplicationController{
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
		}: true,

		{
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
				&apiv1.ReplicationController{
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
		}: false,
	}

	for k, v := range testCases {
		assert.Equal(t, v, k.IsExists())
	}
}
