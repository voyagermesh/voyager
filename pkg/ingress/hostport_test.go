package ingress

import (
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.RoleBinding{
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
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fake-foo",
							Namespace: "bar",
						},
					},

					&core.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.RoleBinding{
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
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fake-foo",
							Namespace: "bar",
						},
					},

					&core.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.RoleBinding{
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
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "fake-foo",
							Namespace: "bar",
						},
					},

					&core.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.RoleBinding{
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
					&apps.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&core.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      api.VoyagerPrefix + "foo",
							Namespace: "bar",
						},
					},

					&rbac.RoleBinding{
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
		t.Log(k.Ingress.Name, k.Ingress.Annotations)
		assert.Equal(t, v, k.IsExists())
	}
}
