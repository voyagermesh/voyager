/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"testing"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

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
