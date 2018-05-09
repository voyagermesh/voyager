package ingress

import (
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestALPNOptions(t *testing.T) {
	dataTable := map[string][]string{
		"": {},
		"alpn h2,http/1.0,http/1.1": {
			"h2",
			"http/1.1",
			"http/1.0",
		},
	}

	for k, v := range dataTable {
		tcpRule := api.TCPIngressRuleValue{ALPN: v}
		assert.Equal(t, k, tcpRule.ParseALPNOptions())
	}
}

func TestConvertRulesForSSLPassthrough(t *testing.T) {
	c := &controller{
		Ingress: &api.Ingress{},
		cfg: config.Config{
			CloudProvider: "minikube",
		},
	}
	var err error
	for k, result := range dataIng {
		c.Ingress, err = api.NewEngressFromIngress(k)
		if assert.Equal(t, err, nil, "Ingress/Migrate:", k.Name, result, err) {
			err = c.convertRulesForSSLPassthrough()
			assert.Equal(t, err == nil, result, "Ingress:", k.Name, result, err)
		}
	}
	for k, result := range dataEng {
		c.Ingress = k
		err = c.convertRulesForSSLPassthrough()
		assert.Equal(t, err == nil, result, "Engress:", k.Name, result, err)
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
	}: false,
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
	}: false,
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
				{
					Host: "not-tls-host",
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
	}: false, // can't use TLS in passthrough mode
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-5", Annotations: sslPassthroughAnnotation},
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
	}: true,
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
	}: false,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-3", Annotations: sslPassthroughAnnotation},
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
	}: false,
	{
		ObjectMeta: metav1.ObjectMeta{Name: "data-4", Annotations: sslPassthroughAnnotation},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					SecretName: "voyager-cert",
					Hosts: []string{
						"tls-host",
					},
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: "tls-host",
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
				{
					Host: "not-tls-host",
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
	}: false, // can't use TLS in passthrough mode
}
