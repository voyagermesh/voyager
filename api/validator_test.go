package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIsValid(t *testing.T) {
	for k, result := range dataTables {
		assert.Equal(t, k.IsValid("") == nil, result)
	}
}

var dataTables = map[*Ingress]bool{
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
						},
					},
				},
			},
		},
	}: false,
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Port: intstr.FromInt(80),
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(80),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(80),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					Host: "this-is-host-one",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Port: intstr.FromInt(80),
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "this-is-host-one",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Port: intstr.FromInt(80),
							Paths: []HTTPIngressPath{
								{
									Path: "/should-be-true",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
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
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					Host: "this-is-host-one",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Port: intstr.FromInt(80),
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "this-is-host-one",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Port: intstr.FromInt(8091),
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(3444),
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
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
							Backend: IngressBackend{
								ServiceName: "foo2",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3435),
							Backend: IngressBackend{
								ServiceName: "foo2",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: true,
	{
		Spec: IngressSpec{
			Backend: &HTTPIngressBackend{
				IngressBackend: IngressBackend{
					ServiceName: "foo",
					ServicePort: intstr.FromString("80"),
				}},
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/test-dns",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										}},
								},
								{
									Path: "/test-no-dns",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										}},
								},
								{
									Path: "/test-no-backend-redirect",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
										}},
								},
								{
									Path: "/test-no-backend-rule-redirect",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("80"),
											BackendRule: []string{
												"http-request redirect location https://google.com code 302",
											},
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/redirect-rule",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											BackendRule: []string{
												"http-request redirect location https://github.com/appscode/discuss/issues code 301",
											},
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
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/redirect",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
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
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/back-end",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromString("8989"),
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
}
