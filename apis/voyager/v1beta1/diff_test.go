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
		"voyager": true,
		"":        true,
		"gce":     true,
		"gke":     true,
		"aws":     true,
	}

	dataTables[&Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				APISchema:                 APISchemaIngress,
				ingressClassAnnotationKey: "voyager",
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
				APISchema:                 APISchemaIngress,
				ingressClassAnnotationKey: ProviderGCE,
			},
		},
	}] = map[string]bool{
		"voyager": false,
		"":        false,
		"gce":     true,
		"gke":     false,
		"aws":     false,
	}

	dataTables[&Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				APISchema: APISchemaIngress,
			},
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
