package ingress

import (
	"testing"
	api "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
apiv1 "k8s.io/client-go/pkg/api/v1"
extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestResourceIsExists(t *testing.T) {
	testCases := map[*EngressController]bool{
		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,
		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeLoadBalancer,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&apiv1.ReplicationController{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: apiv1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeLoadBalancer,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&apiv1.ReplicationController{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "fakefoo",
						Namespace: "bar",
					},
				},

				&apiv1.Service{
					ObjectMeta: apiv1.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&apiv1.ConfigMap{
					ObjectMeta: apiv1.ObjectMeta{
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
