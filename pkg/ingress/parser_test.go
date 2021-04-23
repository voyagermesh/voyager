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
	"voyagermesh.dev/voyager/pkg/config"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestALPNOptions(t *testing.T) {
	{
		dataTable := map[string][]string{
			"": {},
			"alpn h2,http/1.1,http/1.0": {
				"h2",
				"http/1.1",
				"http/1.0",
			},
		}
		for k, v := range dataTable {
			tcpRule := api.IngressRule{
				IngressRuleValue: api.IngressRuleValue{
					TCP: &api.TCPIngressRuleValue{ALPN: v},
				},
			}
			assert.Equal(t, k, tcpRule.ParseALPNOptions())
		}
	}
	{
		dataTable := map[string][]string{
			"alpn http/1.1": {},
			"alpn h2,http/1.1,http/1.0": {
				"h2",
				"http/1.1",
				"http/1.0",
			},
		}
		for k, v := range dataTable {
			httpRule := api.IngressRule{
				IngressRuleValue: api.IngressRuleValue{
					HTTP: &api.HTTPIngressRuleValue{ALPN: v},
				},
			}
			assert.Equal(t, k, httpRule.ParseALPNOptions())
		}
	}
}

func TestConvertRulesForSSLPassthrough(t *testing.T) {
	c := &controller{
		Ingress: &api.Ingress{},
		cfg: config.Config{
			CloudProvider: api.ProviderMinikube,
		},
	}
	var err error
	for k, result := range dataIng {
		c.Ingress, err = api.NewEngressFromIngress(k)
		if assert.Equal(t, err, nil, "Ingress/Migrate: %v, result: %v, reason: %v", k.Name, result, err) {
			err = c.convertRulesForSSLPassthrough()
			assert.Equal(t, err == nil, result, "Ingress: %v, result: %v, reason: %v", k.Name, result, err)
		}
	}
	for k, result := range dataEng {
		c.Ingress = k
		err = c.convertRulesForSSLPassthrough()
		assert.Equal(t, err == nil, result, "Engress: %v, result: %v, reason: %v", k.Name, result, err)
	}
}

var sslPassthroughAnnotation = map[string]string{api.SSLPassthrough: "true"}

var dataEng = map[*api.Ingress]bool{
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-1", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: true,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-2", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
								{
									Path: "/path-2",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("8080"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: false, // multiple paths
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-3", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: false, // multiple rules, same port, empty host
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-4", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			TLS: []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Name: "voyager-cert",
					},
					Hosts: []string{
						"tls-host",
					},
				},
			},
			Rules: []api.IngressRule{
				{
					Host: "tls-host",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: true, // when converting http to tcp in ssl-passthrough, always set NoTLS=true, so no validation error
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-5", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			TLS: []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Name: "voyager-cert",
					},
					Hosts: []string{
						"tls-host",
					},
				},
			},
			Rules: []api.IngressRule{
				{
					Host: "tls-host",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4545),
							Backend: api.IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromString("80"),
							},
						},
					},
				},
			},
		},
	}: false, // tls specified for tcp rule
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-6", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			TLS: []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Name: "voyager-cert",
					},
					Hosts: []string{
						"tls-host",
					},
				},
			},
			Rules: []api.IngressRule{
				{
					Host: "tls-host",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							NoTLS: true,
							Port:  intstr.FromInt(4545),
							Backend: api.IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromString("80"),
							},
						},
					},
				},
			},
		},
	}: true, // tls specified for tcp rule, but found NoTLS
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-7", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Port: intstr.FromInt(4545),
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: true, // different ports
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-8", Annotations: sslPassthroughAnnotation},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					Host: "host-1",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "host-2",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}: true, // tcp-sni case after conversion, same port, multiple hosts
}

var dataIng = map[*v1beta1.Ingress]bool{
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-1", Annotations: sslPassthroughAnnotation},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: v1beta1.IngressBackend{
										ServiceName: "foo",
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
			},
		},
	}: true,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-2", Annotations: sslPassthroughAnnotation},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/path-1",
									Backend: v1beta1.IngressBackend{
										ServiceName: "foo",
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/path-2",
									Backend: v1beta1.IngressBackend{
										ServiceName: "foo",
										ServicePort: intstr.FromString("8080"),
									},
								},
							},
						},
					},
				},
			},
		},
	}: false, // multiple paths
}
