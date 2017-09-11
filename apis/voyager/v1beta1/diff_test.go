package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestShouldHandleIngress(t *testing.T) {
	dataTables := make(map[*Ingress]map[string]bool)

	dataTables[&Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "voyager",
			},
		},
	}] = map[string]bool{
		"voyager": true,
		"":        true,
		"gce":     true,
		"gke":     true,
		"aws":     true,
	}

	dataTables[&Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				APISchema: APISchemaEngress,
			},
		},
	}] = map[string]bool{
		"voyager": true,
		"":        true,
		"gce":     true,
		"gke":     true,
		"aws":     true,
	}

	dataTables[&Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}] = map[string]bool{
		"voyager": false,
		"":        true,
		"gce":     false,
		"gke":     false,
		"aws":     false,
	}

	for k, v := range dataTables {
		for ingressClass, result := range v {
			if !assert.Equal(t, k.ShouldHandleIngress(ingressClass), result) {
				t.Log("Ingress", k.Annotations, "IngressClass", ingressClass, "Expected", result)
			}
		}
	}
}

func TestIsPortChanged(t *testing.T) {
	old := &Ingress{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						HTTP: &HTTPIngressRuleValue{
							Paths: []HTTPIngressPath{
								{
									Path: "/test",
									Backend: HTTPIngressBackend{
										IngressBackend: IngressBackend{
											ServiceName: "foo",
											ServicePort: intstr.FromInt(8080),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	new := &Ingress{
		Spec: IngressSpec{
			Rules: []IngressRule{
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromString("4545"),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromString("4545"),
							},
						},
					},
				},
				{
					IngressRuleValue: IngressRuleValue{
						TCP: &TCPIngressRuleValue{
							Port: intstr.FromString("4949"),
							Backend: IngressBackend{
								ServiceName: "foo",
								ServicePort: intstr.FromString("4545"),
							},
						},
					},
				},
			},
		},
	}

	assert.True(t, old.IsPortChanged(*new, ""))

}
