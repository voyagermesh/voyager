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

package framework

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	api_v1beta1 "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	"gomodules.xyz/pointer"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func (ni *ingressInvocation) GetSkeleton() *api_v1beta1.Ingress {
	ing := &api_v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ni.UniqueName(),
			Namespace: ni.Namespace(),
			Annotations: map[string]string{
				api_v1beta1.DefaultsTimeOut: `{"connect": "5s", "server": "10s"}`,
			},
		},
		Spec: api_v1beta1.IngressSpec{
			Rules:         make([]api_v1beta1.IngressRule, 0),
			FrontendRules: make([]api_v1beta1.FrontendRule, 0),
		},
	}

	return ing
}

func (ni *ingressInvocation) SetSkeletonRule(ing *api_v1beta1.Ingress) {
	ing.Spec.Rules = []api_v1beta1.IngressRule{
		{
			IngressRuleValue: api_v1beta1.IngressRuleValue{
				HTTP: &api_v1beta1.HTTPIngressRuleValue{
					Paths: []api_v1beta1.HTTPIngressPath{
						{
							Path: "/testpath",
							Backend: api_v1beta1.HTTPIngressBackend{
								IngressBackend: api_v1beta1.IngressBackend{
									ServiceName: ni.TestServerName(),
									ServicePort: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (ni *ingressInvocation) SetSkeletonFrontendRule(ing *api_v1beta1.Ingress) {
	ing.Spec.FrontendRules = []api_v1beta1.FrontendRule{
		{
			Port:  intstr.FromInt(3124),
			Rules: []string{"timeout client 60s"},
		},
	}
}

func (ni *ingressInvocation) SetDaemonSkeletonRule(ing *api_v1beta1.Ingress) {
	ing.Annotations = map[string]string{
		api_v1beta1.LBType: api_v1beta1.LBTypeHostPort,
	}
	ing.Spec.NodeSelector = ni.NodeSelector()
	ing.Spec.Rules = []api_v1beta1.IngressRule{
		{
			IngressRuleValue: api_v1beta1.IngressRuleValue{
				HTTP: &api_v1beta1.HTTPIngressRuleValue{
					Paths: []api_v1beta1.HTTPIngressPath{
						{
							Path: "/testpath",
							Backend: api_v1beta1.HTTPIngressBackend{
								IngressBackend: api_v1beta1.IngressBackend{
									ServiceName: ni.TestServerName(),
									ServicePort: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (ni *ingressInvocation) setupTestServers() error {
	if err := ni.createTestServerController(); err != nil {
		return err
	}

	if err := ni.createTestServerService(); err != nil {
		return err
	}

	if err := ni.createEmptyService(); err != nil {
		return err
	}
	return nil
}

func (ni *ingressInvocation) createTestServerController() error {
	_, err := ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: ni.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + ni.app,
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-server-" + ni.app,
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-server-" + ni.app,
					},
				},
				Spec: ni.testServerPodSpec(),
			},
		},
	}, metav1.CreateOptions{})
	return err
}

func (ni *ingressInvocation) createTestServerService() error {
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: ni.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + ni.app,
			},
		},
		Spec: core.ServiceSpec{
			Ports: ni.testServerServicePorts(),
			Selector: map[string]string{
				"app": "test-server-" + ni.app,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerHTTPSResourceName,
			Namespace: ni.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + ni.app,
			},
			Annotations: map[string]string{
				"ingress.appscode.com/backend-tls": "ssl verify none",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       443,
					TargetPort: intstr.FromInt(6443),
					Protocol:   "TCP",
				},
				{
					Name:       "http-2",
					Port:       3443,
					TargetPort: intstr.FromInt(3443),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "test-server-" + ni.app,
			},
		},
	}, metav1.CreateOptions{})
	return err
}

func (ni *ingressInvocation) createEmptyService() error {
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      emptyServiceName,
			Namespace: ni.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + ni.app,
			},
		},
		Spec: core.ServiceSpec{
			Ports: ni.testServerServicePorts(),
			Selector: map[string]string{
				"invalid": "test-server-" + ni.app,
			},
		},
	}, metav1.CreateOptions{})
	return err
}

func (ni *ingressInvocation) testServerPodSpec() core.PodSpec {
	return core.PodSpec{
		Containers: []core.Container{
			{
				Name:  "server",
				Image: testServerImage,
				Env: []core.EnvVar{
					{
						Name: "POD_NAME",
						ValueFrom: &core.EnvVarSource{
							FieldRef: &core.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
				},
				Ports: []core.ContainerPort{
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
						Name:          "https-1",
						ContainerPort: 6443,
					},
					{
						Name:          "https-2",
						ContainerPort: 3443,
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
					{
						Name:          "proxy",
						ContainerPort: 6767,
					},
				},
			},
		},
	}
}

func (ni *ingressInvocation) testServerServicePorts() []core.ServicePort {
	return []core.ServicePort{
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
		{
			Name:       "proxy",
			Port:       6767,
			TargetPort: intstr.FromInt(6767),
			Protocol:   "TCP",
		},
	}
}

func (ni *ingressInvocation) waitForTestServer() error {
	var err error
	var ep *core.Endpoints
	for it := 0; it < MaxRetry; it++ {
		ep, err = ni.KubeClient.CoreV1().Endpoints(ni.Namespace()).Get(context.TODO(), testServerResourceName, metav1.GetOptions{})
		if err == nil {
			if len(ep.Subsets) > 0 {
				if len(ep.Subsets[0].Addresses) > 0 {
					break
				}
			}
		}
		klog.Infoln("Waiting for TestServer to be ready")
		time.Sleep(time.Second * 5)
	}
	return err
}

func (ni *ingressInvocation) NodeSelector() map[string]string {
	if ni.Config.CloudProvider == api.ProviderMinikube {
		return map[string]string{"kubernetes.io/hostname": "minikube"}
	}
	klog.Warningln("No node selector provided for daemon ingress")
	return map[string]string{}
}

func (ni *ingressInvocation) getMinikubeURLs(k kubernetes.Interface, ing *api_v1beta1.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)

	// get offshoot service
	var svc *core.Service
	err := wait.PollImmediate(2*time.Second, 3*time.Minute, func() (bool, error) {
		var err error
		svc, err = k.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false, nil // retry
		} else {
			return true, nil
		}
	})
	if err != nil { // failed even after retry
		return nil, err
	}

	for _, p := range svc.Spec.Ports {
		var accessPort int32
		if svc.Spec.Type == core.ServiceTypeClusterIP {
			accessPort = p.Port
		} else {
			accessPort = p.NodePort
		}
		if accessPort > 0 {
			u, err := url.Parse(fmt.Sprintf("http://%s:%d", ni.MinikubeIP(), accessPort))
			if err != nil {
				return nil, err
			}
			serverAddr = append(serverAddr, u.String())
		}
	}
	return serverAddr, nil
}

func (ni *ingressInvocation) getLoadBalancerURLs(ing *api_v1beta1.Ingress) ([]string, error) {
	if ni.Config.CloudProvider == api.ProviderMinikube {
		return ni.getMinikubeURLs(ni.KubeClient, ing)
	}

	var serverAddr []string

	svc, err := ni.GetServiceWithLoadBalancerStatus(ing.OffshootName(), ing.Namespace)
	Expect(err).NotTo(HaveOccurred())

	ips := make([]string, 0)
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ni.Config.CloudProvider == api.ProviderAWS {
			ips = append(ips, ingress.Hostname)
		} else {
			ips = append(ips, ingress.IP)
		}
	}

	var ports []int32
	if len(svc.Spec.Ports) > 0 {
		for _, port := range svc.Spec.Ports {
			if port.NodePort > 0 {
				ports = append(ports, port.Port)
			}
		}
	}

	for _, port := range ports {
		for _, ip := range ips {
			u, err := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
			if err != nil {
				return nil, err
			}
			serverAddr = append(serverAddr, u.String())
		}
	}

	return serverAddr, nil
}

func (ni *ingressInvocation) getHostPortURLs(ing *api_v1beta1.Ingress) ([]string, error) {
	if ni.Config.CloudProvider == api.ProviderMinikube {
		return ni.getMinikubeURLs(ni.KubeClient, ing)
	}

	var (
		svc        *core.Service
		ports      []int32
		serverAddr []string
		err        error
	)

	Eventually(func() error {
		svc, err = ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Spec.Ports) > 0 {
				for _, port := range svc.Spec.Ports {
					ports = append(ports, port.Port)
				}
			}
			return nil
		}
		return err
	}, "10m", "10s").Should(BeNil())
	if err != nil {
		return nil, err
	}

	pods, err := ni.KubeClient.CoreV1().Pods(ing.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		node, err := ni.KubeClient.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for _, addr := range node.Status.Addresses {
			if addr.Type == core.NodeExternalIP {
				for _, port := range ports {
					u, err := url.Parse(fmt.Sprintf("http://%s:%d", addr.Address, port))
					if err != nil {
						return nil, err
					}
					serverAddr = append(serverAddr, u.String())
				}
			}
		}
	}

	return serverAddr, nil
}

func (ni *ingressInvocation) getNodePortURLs(ing *api_v1beta1.Ingress) ([]string, error) {
	if ni.Config.CloudProvider == api.ProviderMinikube {
		return ni.getMinikubeURLs(ni.KubeClient, ing)
	}

	serverAddr := make([]string, 0)
	nodes, err := ni.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(ing.Spec.NodeSelector).String(),
	})
	if err != nil {
		return nil, err
	}

	var svc *core.Service
	var ports []int32

	Eventually(func() error {
		svc, err = ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Spec.Ports) > 0 {
				for _, port := range svc.Spec.Ports {
					ports = append(ports, port.NodePort)
				}
			}
			return nil
		}
		return err
	}, "10m", "10s").Should(BeNil())
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == core.NodeExternalIP {
				for _, port := range ports {
					u, err := url.Parse(fmt.Sprintf("http://%s:%d", addr.Address, port))
					if err != nil {
						return nil, err
					}
					serverAddr = append(serverAddr, u.String())
				}
			}
		}
	}

	return serverAddr, nil
}

func (ni *ingressInvocation) CheckTestServersPortAssignments(ing *api_v1beta1.Ingress) error {
	ni.Lock.Lock()
	defer ni.Lock.Unlock()

	dep, err := ni.KubeClient.AppsV1().Deployments(ni.TestNamespace).Get(context.TODO(), ni.TestServerName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	svc, err := ni.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Removing pods so that endpoints get updated
	dep.Spec.Replicas = pointer.Int32P(0)
	_, err = ni.KubeClient.AppsV1().Deployments(dep.Namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	sel, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return err
	}

	for {

		pods, _ := ni.KubeClient.CoreV1().Pods(dep.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: sel.String(),
		})
		if len(pods.Items) <= 0 {
			break
		}
		time.Sleep(time.Second * 1)
	}

	svcUpdated, err := ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, oldPort := range svc.Spec.Ports {
		for _, newPort := range svcUpdated.Spec.Ports {
			if oldPort.Port == newPort.Port {
				if oldPort.NodePort != newPort.NodePort {
					return errors.New("NodePort Mismatched")
				}
			}
		}
	}

	dep, err = ni.KubeClient.AppsV1().Deployments(ni.TestNamespace).Get(context.TODO(), ni.TestServerName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	dep.Spec.Replicas = pointer.Int32P(2)
	_, err = ni.KubeClient.AppsV1().Deployments(dep.Namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	svcUpdated, err = ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, oldPort := range svc.Spec.Ports {
		for _, newPort := range svcUpdated.Spec.Ports {
			if oldPort.Port == newPort.Port {
				if oldPort.NodePort != newPort.NodePort {
					return errors.New("NodePort Mismatched")
				}
			}
		}
	}
	return nil
}

func (ni *ingressInvocation) SupportsServiceIP() bool {
	return ni.Config.CloudProvider == api.ProviderAWS ||
		ni.Config.CloudProvider == api.ProviderGCE ||
		ni.Config.CloudProvider == api.ProviderGKE ||
		ni.Config.CloudProvider == api.ProviderAzure ||
		ni.Config.CloudProvider == "acs" ||
		ni.Config.CloudProvider == "aks" ||
		ni.Config.CloudProvider == "openstack"
}

func (ni *ingressInvocation) CreateResourceWithHostNames() (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      ni.UniqueName(),
		Namespace: ni.Namespace(),
		Labels: map[string]string{
			"app": "test-server-" + ni.app,
			"v":   ni.UniqueName(),
		},
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			ClusterIP: "None",
			Ports: []core.ServicePort{
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
			Selector: meta.Labels,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().StatefulSets(ni.Namespace()).Create(context.TODO(), &apps.StatefulSet{
		ObjectMeta: meta,
		Spec: apps.StatefulSetSpec{
			Replicas:    pointer.Int32P(2),
			ServiceName: meta.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: meta.Labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: meta.Labels,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
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
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func (ni *ingressInvocation) DeleteResourceWithHostNames(meta metav1.ObjectMeta) error {
	policy := metav1.DeletePropagationBackground
	if err := ni.KubeClient.AppsV1().StatefulSets(meta.Namespace).Delete(context.TODO(), meta.Name, metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		return err
	}

	if err := ni.KubeClient.CoreV1().Services(meta.Namespace).Delete(context.TODO(), meta.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (ni *ingressInvocation) CreateResourceWithBackendWeight() (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      ni.UniqueName(),
		Namespace: ni.Namespace(),
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "deployment",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v1",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v1",
					},
					Annotations: map[string]string{
						api_v1beta1.BackendWeight: "90",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-2-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v2",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v2",
					},
					Annotations: map[string]string{
						api_v1beta1.BackendWeight: "10",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (ni *ingressInvocation) DeleteResourceWithBackendWeight(meta metav1.ObjectMeta) {
	deletePolicy := metav1.DeletePropagationForeground

	dp1, err := ni.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), "dep-1-"+meta.Name, metav1.GetOptions{})
	if err == nil {
		Expect(ni.KubeClient.AppsV1().Deployments(dp1.Namespace).Delete(context.TODO(), dp1.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})).NotTo(HaveOccurred())
	}

	dp2, err := ni.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), "dep-2-"+meta.Name, metav1.GetOptions{})
	if err == nil {
		Expect(ni.KubeClient.AppsV1().Deployments(dp2.Namespace).Delete(context.TODO(), dp2.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})).NotTo(HaveOccurred())
	}

	Expect(ni.KubeClient.CoreV1().Services(meta.Namespace).Delete(context.TODO(), meta.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})).NotTo(HaveOccurred())
}

func (ni *ingressInvocation) CreateResourceWithBackendMaxConn(maxconn int) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      ni.UniqueName(),
		Namespace: ni.Namespace(),
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "deployment",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v1",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v1",
					},
					Annotations: map[string]string{
						api_v1beta1.MaxConnections: strconv.Itoa(maxconn),
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (ni *ingressInvocation) CreateResourceWithServiceAuth(secret *core.Secret) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      ni.UniqueName(),
		Namespace: ni.Namespace(),
		Annotations: map[string]string{
			api_v1beta1.AuthType:   "basic",
			api_v1beta1.AuthRealm:  "Realm returned",
			api_v1beta1.AuthSecret: secret.Name,
		},
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "deployment",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v1",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v1",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (ni *ingressInvocation) CreateResourceWithServiceAnnotation(svcAnnotation map[string]string) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:        ni.UniqueName(),
		Namespace:   ni.Namespace(),
		Annotations: svcAnnotation,
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
				{
					Name:       "agent-check-tcp",
					Port:       5555,
					TargetPort: intstr.FromInt(5555),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": meta.Name,
			},
			Type: core.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         meta.Name,
					"app-version": "v1",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         meta.Name,
						"app-version": "v1",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
								{
									Name:          "agent-check-tcp",
									ContainerPort: 5555,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})

	return meta, err
}

func (ni *ingressInvocation) CreateResourceWithSendProxy(version string) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      ni.UniqueName(),
		Namespace: ni.Namespace(),
		Annotations: map[string]string{
			api_v1beta1.SendProxy: version,
		},
	}
	_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Create(context.TODO(), &core.Service{
		ObjectMeta: meta,
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "proxy",
					Port:       6767,
					TargetPort: intstr.FromInt(6767),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": meta.Name,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return meta, err
	}

	_, err = ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Create(context.TODO(), &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: pointer.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         meta.Name,
					"app-version": "v1",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         meta.Name,
						"app-version": "v1",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "server",
							Image: testServerImage,
							Env: []core.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          "proxy",
									ContainerPort: 6767,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})

	return meta, err
}

func (ni *ingressInvocation) CreateTLSSecretForHost(name string, hosts []string) (*core.Secret, error) {
	crt, key, err := ni.CertStore.NewServerCertPairBytes(
		cert.AltNames{
			DNSNames: append([]string{"server"}, hosts...),
			IPs:      []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.99.100")},
		},
	)
	if err != nil {
		return nil, err
	}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ni.Namespace(),
		},
		Type: core.SecretTypeTLS,
		Data: map[string][]byte{
			core.TLSCertKey:       crt,
			core.TLSPrivateKeyKey: key,
		},
	}
	return ni.KubeClient.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
}

func (ni *ingressInvocation) GetIngressWithLoadBalancerStatus(name, namespace string) (*api_v1beta1.Ingress, error) {
	var (
		ing *api_v1beta1.Ingress
		err error
	)
	err = wait.PollImmediate(2*time.Second, 20*time.Minute, func() (bool, error) {
		ing, err = ni.VoyagerClient.VoyagerV1beta1().Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil || len(ing.Status.LoadBalancer.Ingress) == 0 { // retry
			return false, nil
		} else {
			return true, nil
		}
	})
	if err != nil {
		return nil, errors.Errorf("failed to get Status.LoadBalancer.Ingress for ingress %s/%s", name, namespace)
	}
	return ing, nil
}

func (ni *ingressInvocation) GetServiceWithLoadBalancerStatus(name, namespace string) (*core.Service, error) {
	var (
		svc *core.Service
		err error
	)
	err = wait.PollImmediate(2*time.Second, 20*time.Minute, func() (bool, error) {
		svc, err = ni.KubeClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil || len(svc.Status.LoadBalancer.Ingress) == 0 { // retry
			return false, nil
		} else {
			return true, nil
		}
	})
	if err != nil {
		return nil, errors.Errorf("failed to get Status.LoadBalancer.Ingress for service %s/%s", name, namespace)
	}
	return svc, nil
}

func (ni *ingressInvocation) GetNodePortServiceURLForSpecificPort(svcName string, port int32) (string, error) {

	var err error

	nodeList, err := ni.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	node := nodeList.Items[0]

	var nodeIP string
	if node.Name == api.ProviderMinikube {
		nodeIP = ni.MinikubeIP()
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			nodeIP = addr.Address
		}
	}

	// get the k8s core.service
	var svc *core.Service
	Eventually(func() error {
		svc, err = ni.KubeClient.CoreV1().Services(ni.TestNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
		return err
	}, "10m", "10s").Should(BeNil())

	for _, svcPort := range svc.Spec.Ports {
		if svcPort.Port == port {
			u, err := url.Parse(fmt.Sprintf("http://%s:%d", nodeIP, svcPort.NodePort))
			if err != nil {
				return "", err
			}
			return u.String(), nil
		}
	}

	return "", errors.New("Port not found\n")
}
