package e2e

import (
	"github.com/appscode/voyager/test/testframework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var testServerSvc = &apiv1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-server",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "test-server",
		},
	},
	Spec: apiv1.ServiceSpec{
		Ports: []apiv1.ServicePort{
			{
				Name:       "http-1",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
				Protocol:   "TCP",
			},
			{
				Name:       "http-2",
				Port:       8989,
				TargetPort: intstr.FromInt(8989),
				Protocol:   "TCP",
			},
			{
				Name:       "http-3",
				Port:       9090,
				TargetPort: intstr.FromInt(9090),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-1",
				Port:       4343,
				TargetPort: intstr.FromInt(4343),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-2",
				Port:       4545,
				TargetPort: intstr.FromInt(4545),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-3",
				Port:       5656,
				TargetPort: intstr.FromInt(5656),
				Protocol:   "TCP",
			},
		},
		Selector: map[string]string{
			"app":  "test-server",
			"name": "test-rc",
		},
	},
}

var testServerRc = &apiv1.ReplicationController{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-server",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app":  "test-server",
			"name": "test-rc",
		},
	},
	Spec: apiv1.ReplicationControllerSpec{
		Replicas: 2,
		Selector: map[string]string{
			"app": "test-server",
		},
		Template: &apiv1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":  "test-server",
					"name": "test-rc",
				},
			},
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Name:  "server",
						Image: "appscode/test-server:1.4",
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
		Ports: []apiv1.ServicePort{
			{
				Name:       "http-1",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
				Protocol:   "TCP",
			},
			{
				Name:       "http-2",
				Port:       8989,
				TargetPort: intstr.FromInt(8989),
				Protocol:   "TCP",
			},
			{
				Name:       "http-3",
				Port:       9090,
				TargetPort: intstr.FromInt(9090),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-1",
				Port:       4343,
				TargetPort: intstr.FromInt(4343),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-2",
				Port:       4545,
				TargetPort: intstr.FromInt(4545),
				Protocol:   "TCP",
			},
			{
				Name:       "tcp-3",
				Port:       5656,
				TargetPort: intstr.FromInt(5656),
				Protocol:   "TCP",
			},
		},
		Selector: map[string]string{
			"app":  "e2e-test",
			"name": "test-ss",
		},
	},
}

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
		Replicas:    3,
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
