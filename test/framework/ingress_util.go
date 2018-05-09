package framework

import (
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	api_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
)

func (i *ingressInvocation) GetSkeleton() *api_v1beta1.Ingress {
	ing := &api_v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.UniqueName(),
			Namespace: i.Namespace(),
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

func (i *ingressInvocation) SetSkeletonRule(ing *api_v1beta1.Ingress) {
	ing.Spec.Rules = []api_v1beta1.IngressRule{
		{
			IngressRuleValue: api_v1beta1.IngressRuleValue{
				HTTP: &api_v1beta1.HTTPIngressRuleValue{
					Paths: []api_v1beta1.HTTPIngressPath{
						{
							Path: "/testpath",
							Backend: api_v1beta1.HTTPIngressBackend{
								IngressBackend: api_v1beta1.IngressBackend{
									ServiceName: i.TestServerName(),
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

func (i *ingressInvocation) SetSkeletonFrontendRule(ing *api_v1beta1.Ingress) {
	ing.Spec.FrontendRules = []api_v1beta1.FrontendRule{
		{
			Port:  intstr.FromInt(3124),
			Rules: []string{"timeout client 60s"},
		},
	}
}

func (i *ingressInvocation) SetDaemonSkeletonRule(ing *api_v1beta1.Ingress) {
	ing.Annotations = map[string]string{
		api_v1beta1.LBType: api_v1beta1.LBTypeHostPort,
	}
	ing.Spec.NodeSelector = i.NodeSelector()
	ing.Spec.Rules = []api_v1beta1.IngressRule{
		{
			IngressRuleValue: api_v1beta1.IngressRuleValue{
				HTTP: &api_v1beta1.HTTPIngressRuleValue{
					Paths: []api_v1beta1.HTTPIngressPath{
						{
							Path: "/testpath",
							Backend: api_v1beta1.HTTPIngressBackend{
								IngressBackend: api_v1beta1.IngressBackend{
									ServiceName: i.TestServerName(),
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

func (i *ingressInvocation) setupTestServers() error {
	if err := i.createTestServerController(); err != nil {
		return err
	}

	if err := i.createTestServerService(); err != nil {
		return err
	}
	return nil
}

func (i *ingressInvocation) createTestServerController() error {
	_, err := i.KubeClient.CoreV1().ReplicationControllers(i.Namespace()).Create(&core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + i.app,
			},
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(2),
			Selector: map[string]string{
				"app": "test-server-" + i.app,
			},
			Template: &core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-server-" + i.app,
					},
				},
				Spec: i.testServerPodSpec(),
			},
		},
	})
	return err
}

func (i *ingressInvocation) createTestServerService() error {
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + i.app,
			},
		},
		Spec: core.ServiceSpec{
			Ports: i.testServerServicePorts(),
			Selector: map[string]string{
				"app": "test-server-" + i.app,
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerHTTPSResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + i.app,
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
				"app": "test-server-" + i.app,
			},
		},
	})
	return err
}

func (i *ingressInvocation) testServerPodSpec() core.PodSpec {
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

func (i *ingressInvocation) testServerServicePorts() []core.ServicePort {
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

func (i *ingressInvocation) waitForTestServer() error {
	var err error
	var ep *core.Endpoints
	for it := 0; it < MaxRetry; it++ {
		ep, err = i.KubeClient.CoreV1().Endpoints(i.Namespace()).Get(testServerResourceName, metav1.GetOptions{})
		if err == nil {
			if len(ep.Subsets) > 0 {
				if len(ep.Subsets[0].Addresses) > 0 {
					break
				}
			}
		}
		log.Infoln("Waiting for TestServer to be ready")
		time.Sleep(time.Second * 5)
	}
	return err
}

func (i *ingressInvocation) NodeSelector() map[string]string {
	if i.Operator.CloudProvider == "minikube" {
		return map[string]string{"kubernetes.io/hostname": "minikube"}
	}
	log.Warningln("No node selector provided for daemon ingress")
	return map[string]string{}
}

func getMinikubeIP() (ip net.IP, err error) {
	wait.PollImmediate(2*time.Second, 3*time.Minute, func() (bool, error) {
		var outputs []byte
		if outputs, err = exec.Command("minikube", "ip").CombinedOutput(); err != nil {
			return false, nil // retry
		} else {
			output := strings.TrimSpace(string(outputs))
			if ip = net.ParseIP(output); ip == nil {
				err = errors.Errorf("failed to parse minikube ip: %s", output)
				return false, nil // retry
			} else {
				return true, nil
			}
		}
	})
	return
}

func getMinikubeURLs(k kubernetes.Interface, ing *api_v1beta1.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)

	minikubeIP, err := getMinikubeIP()
	if err != nil {
		return nil, err
	}

	// get offshoot service
	var svc *core.Service
	wait.PollImmediate(2*time.Second, 3*time.Minute, func() (bool, error) {
		svc, err = k.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
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
			u, err := url.Parse(fmt.Sprintf("http://%s:%d", minikubeIP, accessPort))
			if err != nil {
				return nil, err
			}
			serverAddr = append(serverAddr, u.String())
		}
	}
	return serverAddr, nil
}

func getLoadBalancerURLs(provider string, k kubernetes.Interface, ing *api_v1beta1.Ingress) ([]string, error) {
	if provider == "minikube" {
		return getMinikubeURLs(k, ing)
	}

	serverAddr := make([]string, 0)
	var err error

	gomega.Eventually(func() error {
		var svc *core.Service
		svc, err = k.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Status.LoadBalancer.Ingress) != 0 {
				ips := make([]string, 0)
				for _, ingress := range svc.Status.LoadBalancer.Ingress {
					if provider == "aws" {
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
							return err
						}
						serverAddr = append(serverAddr, u.String())
					}
				}
				return nil
			}
		}
		return nil
	}, "5m", "10s").Should(gomega.BeNil())
	if err != nil {
		return nil, err
	}
	return serverAddr, nil
}

func getHostPortURLs(provider string, k kubernetes.Interface, ing *api_v1beta1.Ingress) ([]string, error) {
	if provider == "minikube" {
		return getMinikubeURLs(k, ing)
	}

	serverAddr := make([]string, 0)
	nodes, err := k.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(ing.Spec.NodeSelector).String(),
	})
	if err != nil {
		return nil, err
	}

	var svc *core.Service
	var ports []int32

	gomega.Eventually(func() error {
		svc, err = k.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Spec.Ports) > 0 {
				for _, port := range svc.Spec.Ports {
					ports = append(ports, port.Port)
				}
			}
			return nil
		}
		return err
	}, "10m", "10s").Should(gomega.BeNil())
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

func getNodePortURLs(provider string, k kubernetes.Interface, ing *api_v1beta1.Ingress) ([]string, error) {
	if provider == "minikube" {
		return getMinikubeURLs(k, ing)
	}

	serverAddr := make([]string, 0)
	nodes, err := k.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(ing.Spec.NodeSelector).String(),
	})
	if err != nil {
		return nil, err
	}

	var svc *core.Service
	var ports []int32

	gomega.Eventually(func() error {
		svc, err = k.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Spec.Ports) > 0 {
				for _, port := range svc.Spec.Ports {
					ports = append(ports, port.NodePort)
				}
			}
			return nil
		}
		return err
	}, "10m", "10s").Should(gomega.BeNil())
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

func (i *ingressInvocation) CheckTestServersPortAssignments(ing *api_v1beta1.Ingress) error {
	i.Lock.Lock()
	defer i.Lock.Unlock()

	rc, err := i.KubeClient.CoreV1().ReplicationControllers(i.TestNamespace).Get(i.TestServerName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	svc, err := i.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(ing.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Removing pods so that endpoints get updated
	rc.Spec.Replicas = types.Int32P(0)
	i.KubeClient.CoreV1().ReplicationControllers(rc.Namespace).Update(rc)
	for {
		pods, _ := i.KubeClient.CoreV1().Pods(rc.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(rc.Spec.Selector).String(),
		})
		if len(pods.Items) <= 0 {
			break
		}
		time.Sleep(time.Second * 1)
	}

	svcUpdated, err := i.KubeClient.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
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

	rc, err = i.KubeClient.CoreV1().ReplicationControllers(i.TestNamespace).Get(i.TestServerName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	rc.Spec.Replicas = types.Int32P(2)
	rc, err = i.KubeClient.CoreV1().ReplicationControllers(rc.Namespace).Update(rc)
	if err != nil {
		return err
	}
	svcUpdated, err = i.KubeClient.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
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

func (i *ingressInvocation) SupportsServiceIP() bool {
	return i.Operator.CloudProvider == "aws" ||
		i.Operator.CloudProvider == "gce" ||
		i.Operator.CloudProvider == "gke" ||
		i.Operator.CloudProvider == "azure" ||
		i.Operator.CloudProvider == "acs" ||
		i.Operator.CloudProvider == "openstack"
}

func (i *ingressInvocation) CreateResourceWithHostNames() (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      i.UniqueName(),
		Namespace: i.Namespace(),
		Labels: map[string]string{
			"app": "test-server-" + i.app,
			"v":   i.UniqueName(),
		},
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.AppsV1beta1().StatefulSets(i.Namespace()).Create(&apps.StatefulSet{
		ObjectMeta: meta,
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(2),
			ServiceName: meta.Name,
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
	})
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func (i *ingressInvocation) DeleteResourceWithHostNames(meta metav1.ObjectMeta) error {
	policy := metav1.DeletePropagationBackground
	if err := i.KubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		return err
	}

	if err := i.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (i *ingressInvocation) CreateResourceWithBackendWeight() (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      i.UniqueName(),
		Namespace: i.Namespace(),
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-2-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
	})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (i *ingressInvocation) DeleteResourceWithBackendWeight(meta metav1.ObjectMeta) {
	dp1, err := i.KubeClient.ExtensionsV1beta1().Deployments(meta.Namespace).Get("dep-1-"+meta.Name, metav1.GetOptions{})
	if err == nil {
		dp1.Spec.Replicas = types.Int32P(0)
		i.KubeClient.ExtensionsV1beta1().Deployments(dp1.Namespace).Update(dp1)
	}
	dp2, err := i.KubeClient.ExtensionsV1beta1().Deployments(meta.Namespace).Get("dep-2-"+meta.Name, metav1.GetOptions{})
	if err == nil {
		dp2.Spec.Replicas = types.Int32P(0)
		i.KubeClient.ExtensionsV1beta1().Deployments(dp2.Namespace).Update(dp2)
	}
	time.Sleep(time.Second * 5)
	orphan := false
	i.KubeClient.ExtensionsV1beta1().Deployments(dp1.Namespace).Delete(dp1.Name, &metav1.DeleteOptions{
		OrphanDependents: &orphan,
	})

	i.KubeClient.ExtensionsV1beta1().Deployments(dp2.Namespace).Delete(dp2.Name, &metav1.DeleteOptions{
		OrphanDependents: &orphan,
	})

	i.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{
		OrphanDependents: &orphan,
	})
}

func (i *ingressInvocation) CreateResourceWithBackendMaxConn(maxconn int) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      i.UniqueName(),
		Namespace: i.Namespace(),
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
	})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (i *ingressInvocation) CreateResourceWithServiceAuth(secret *core.Secret) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      i.UniqueName(),
		Namespace: i.Namespace(),
		Annotations: map[string]string{
			api_v1beta1.AuthType:   "basic",
			api_v1beta1.AuthRealm:  "Realm returned",
			api_v1beta1.AuthSecret: secret.Name,
		},
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
	})
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (i *ingressInvocation) CreateResourceWithServiceAnnotation(svcAnnotation map[string]string) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:        i.UniqueName(),
		Namespace:   i.Namespace(),
		Annotations: svcAnnotation,
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
				"app": meta.Name,
			},
		},
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
							},
						},
					},
				},
			},
		},
	})

	return meta, err
}

func (i *ingressInvocation) CreateResourceWithSendProxy(version string) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Name:      i.UniqueName(),
		Namespace: i.Namespace(),
		Annotations: map[string]string{
			api_v1beta1.SendProxy: version,
		},
	}
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
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
	})
	if err != nil {
		return meta, err
	}

	_, err = i.KubeClient.ExtensionsV1beta1().Deployments(i.Namespace()).Create(&extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1-" + meta.Name,
			Namespace: meta.Namespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
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
	})

	return meta, err
}

func (i *ingressInvocation) CreateTLSSecretForHost(name string, hosts []string) (*core.Secret, error) {
	crt, key, err := i.CertStore.NewServerCertPair(
		"server",
		cert.AltNames{
			IPs:      []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.99.100")},
			DNSNames: hosts,
		},
	)
	if err != nil {
		return nil, err
	}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: i.Namespace(),
		},
		Type: core.SecretTypeTLS,
		Data: map[string][]byte{
			core.TLSCertKey:       crt,
			core.TLSPrivateKeyKey: key,
		},
	}
	return i.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
}
