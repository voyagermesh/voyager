package e2e

/*
import (
	"github.com/appscode/voyager/test/testframework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var testServerSvc =
}

var rcReplica int32 = 2

var testServerRc =

var testStatefulSetSvc = &apiv1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ss-svc",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "e2e-test",
		},
	},
	Spec: apiv1.ServiceSpec{
		ClusterIP: "None",
		Ports:
		Selector: map[string]string{
			"app":  "e2e-test",
			"name": "test-ss",
		},
	},
}

var statefulsetReplica int32 = 3

var testServerStatefulSet = &apps.StatefulSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-ss",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app":  "e2e-test",
			"name": "test-ss",
		},
	},
	Spec: apps.StatefulSetSpec{
		Replicas:    &statefulsetReplica,
		ServiceName: testStatefulSetSvc.Name,
		Template: apiv1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "e2e-test",
				},
			},
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Name:  "server",
						Image: "appscode/test-server:1.1",
						Env: []apiv1.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &apiv1.EnvVarSource{
									FieldRef: &apiv1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
						},
						Ports: []apiv1.ContainerPort{
							{
								Name:          "http-1",
								ContainerPort: 8080,
							},
							{
								Name:          "http-2",
								ContainerPort: 8989,
							},
							{
								Name:          "http-3",
								ContainerPort: 9090,
							},
							{
								Name:          "tcp-1",
								ContainerPort: 4343,
							},
							{
								Name:          "tcp-2",
								ContainerPort: 4545,
							},
							{
								Name:          "tcp-3",
								ContainerPort: 5656,
							},
						},
					},
				},
			},
		},
	},
}
*/
