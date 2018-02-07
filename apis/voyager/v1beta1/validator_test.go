package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIsValid(t *testing.T) {
	for k, result := range dataTables {
		k.Migrate()
		err := k.IsValid("minikube")
		if !assert.Equal(t, err == nil, result) {
			fmt.Println("Failed Tests:", k.Name, "Reason\n", err)
		}
	}
}

var dataTables = map[*Ingress]bool{
	{
		ObjectMeta: metav1.ObjectMeta{Name: "Empty backend service name"},
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
							Backend: IngressBackend{
								ServiceName: "",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "Invalid backend service name"},
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromInt(3434),
							Backend: IngressBackend{
								ServiceName: ".default",
								ServicePort: intstr.FromInt(3444),
							},
						},
					},
				},
			},
		},
	}: false,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "spec.rule[0] can specify either HTTP or TCP"},
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{},
						TCP:  &TCPIngressRuleValue{},
					},
				},
			},
		},
	}: false,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "No Backend Service For TCP"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "No Listen Port for TCP"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "TCP and HTTP in Same Port specified"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "TCP and HTTP in Same Port not specified"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "HTTP with host and path"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "HTTP with hosts"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "TCP multi in Same Port"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "TCP with different port"},
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
		ObjectMeta: metav1.ObjectMeta{Name: "Multi rule"},
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
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "https://github.com/appscode/voyager/issues/420",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.appscode.com/type": "HostPort",
			},
		},
		Spec: IngressSpec{
			TLS: []IngressTLS{
				{
					SecretName: "voyager-cert",
					Hosts: []string{
						"minicluster.example.com",
					},
				},
			},
			Rules: []IngressRule{
				{
					Host: "minicluster.example.com",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "cluster-nginx",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "minicluster.example.com",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							NoTLS: true,
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "cluster-nginx",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "domain1.com",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "cluster-nginx",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "domain2.com",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "cluster-nginx",
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      "https://github.com/appscode/voyager/pull/768",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.appscode.com/type": "Internal",
			},
		},
		Spec: IngressSpec{
			TLS: []IngressTLS{
				{
					SecretName: "voyager-cert",
					Hosts: []string{
						"*.example.org.dmmy.me",
					},
				},
			},
			Rules: []IngressRule{
				{
					Host: "*.example.org.dmmy.me",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/resources",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "distro-static",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
								{
									Path: "/admin_resources",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "admin-resources",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "*.example.org.dmmy.me",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							NoTLS: true,
							Paths: []HTTPIngressPath{
								{
									Path: "/resources",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "distro-static",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
								{
									Path: "/admin_resources",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "admin-resources",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "example.org.dmmy.me",
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "verify-tls-cert-dmmy-me",
											ServicePort: intstr.FromString("80"),
										},
									},
								},
							},
						},
					},
				},
			},
			Backend: &HTTPIngressBackend{
				IngressBackend: IngressBackend{
					ServiceName: "distro-biz",
					ServicePort: intstr.FromInt(80),
				},
			},
		},
	}: true,
}
