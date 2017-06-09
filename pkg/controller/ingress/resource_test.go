package ingress

import (
	"testing"

	api "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestResourceIsExists(t *testing.T) {
	testCases := map[*EngressController]bool{
		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,
		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeHostPort,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeLoadBalancer,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&kapi.ReplicationController{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Resource: &api.Ingress{
				ObjectMeta: kapi.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Annotations: map[string]string{
						api.LBType: api.LBTypeLoadBalancer,
					},
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&kapi.ReplicationController{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "fakefoo",
						Namespace: "bar",
					},
				},

				&kapi.Service{
					ObjectMeta: kapi.ObjectMeta{
						Name:      api.VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
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
