package ingress

import (
	"testing"

	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestResourceIsExists(t *testing.T) {
	testCases := map[*EngressController]bool{
		{
			Options: &KubeOptions{
				LBType: LBDaemon,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Options: &KubeOptions{
				LBType: LBDaemon,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Options: &KubeOptions{
				LBType: LBDaemon,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Options: &KubeOptions{
				LBType: LBDaemon,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "fake-foo",
						Namespace: "bar",
					},
				},
			),
		}: false,

		{
			Options: &KubeOptions{
				LBType: LBDaemon,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&extensions.DaemonSet{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Options: &KubeOptions{
				LBType: LBLoadBalancer,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&api.ReplicationController{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},
			),
		}: true,

		{
			Options: &KubeOptions{
				LBType: LBLoadBalancer,
			},
			Config: &aci.Ingress{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			KubeClient: fake.NewSimpleClientset(
				&api.ReplicationController{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "fakefoo",
						Namespace: "bar",
					},
				},

				&api.Service{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
						Namespace: "bar",
					},
				},

				&api.ConfigMap{
					ObjectMeta: api.ObjectMeta{
						Name:      VoyagerPrefix + "foo",
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
