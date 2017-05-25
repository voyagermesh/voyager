package e2e

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/util/intstr"
	"github.com/appscode/voyager/test/testframework"
)

var testServerSvc = &api.Service{
	ObjectMeta: api.ObjectMeta{
		Name:      "test-server",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "test-server",
		},
	},
	Spec: api.ServiceSpec{
		Ports: []api.ServicePort{
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
			"app": "test-server",
			"name": "test-rc",
		},
	},
}

var testServerRc = &api.ReplicationController{
	ObjectMeta: api.ObjectMeta{
		Name:      "test-server",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "test-server",
			"name": "test-rc",
		},
	},
	Spec: api.ReplicationControllerSpec{
		Replicas: 2,
		Selector: map[string]string{
			"app": "test-server",
		},
		Template: &api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: map[string]string{
					"app": "test-server",
					"name": "test-rc",
				},
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "server",
						Image: "appscode/test-server:1.4",
						Env: []api.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &api.EnvVarSource{
									FieldRef: &api.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
						},
						Ports: []api.ContainerPort{
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

var testStatefulSetSvc = &api.Service{
	ObjectMeta: api.ObjectMeta{
		Name:      "ss-svc",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "e2e-test",
		},
	},
	Spec: api.ServiceSpec{
		ClusterIP: "None",
		Ports: []api.ServicePort{
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
			"app": "e2e-test",
			"name": "test-ss",
		},
	},
}

var testServerStatefulSet = &apps.StatefulSet{
	ObjectMeta: api.ObjectMeta{
		Name:      "test-ss",
		Namespace: testframework.TestContext.E2EConfigs.TestNamespace,
		Labels: map[string]string{
			"app": "e2e-test",
			"name": "test-ss",
		},
	},
	Spec: apps.StatefulSetSpec{
		Replicas:    3,
		ServiceName: testStatefulSetSvc.Name,
		Template: api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: map[string]string{
					"app": "e2e-test",
				},
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "server",
						Image: "appscode/test-server:1.1",
						Env: []api.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &api.EnvVarSource{
									FieldRef: &api.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
						},
						Ports: []api.ContainerPort{
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
