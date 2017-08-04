package testframework

import (
	"bytes"
	"errors"
	"net/url"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var (
	defaultUrlTemplate = template.Must(template.New("svc-template").Parse("http://{{.IP}}:{{.Port}}"))
)

func (i *ingressInvocation) GetSkeleton() *api.Ingress {
	ing := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.generateName(),
			Namespace: i.Namespace(),
			Annotations: map[string]string{
				api.DefaultsTimeOut: `{"connect": "5s", "server": "10s"}`,
			},
		},
		Spec: api.IngressSpec{Rules: make([]api.IngressRule, 0)},
	}

	return ing
}

func (i *ingressInvocation) SetSkeletonRule(ing *api.Ingress) {
	ing.Spec.Rules = []api.IngressRule{
		{
			IngressRuleValue: api.IngressRuleValue{
				HTTP: &api.HTTPIngressRuleValue{
					Paths: []api.HTTPIngressPath{
						{
							Path: "/testpath",
							Backend: api.IngressBackend{
								ServiceName: i.TestServerName(),
								ServicePort: intstr.FromInt(80),
							},
						},
					},
				},
			},
		},
	}
}

func (i *ingressInvocation) generateName() string {
	return rand.WithUniqSuffix("e2e-test")
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
	_, err := i.KubeClient.CoreV1().ReplicationControllers(i.Namespace()).Create(&apiv1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": i.app,
			},
		},
		Spec: apiv1.ReplicationControllerSpec{
			Replicas: types.Int32P(2),
			Selector: map[string]string{
				"app": i.app,
			},
			Template: &apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": i.app,
					},
				},
				Spec: i.testServerPodSpec(),
			},
		},
	})
	return err
}

func (i *ingressInvocation) createTestServerService() error {
	_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Create(&apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": i.app,
			},
		},
		Spec: apiv1.ServiceSpec{
			Ports: i.testServerServicePorts(),
			Selector: map[string]string{
				"app": i.app,
			},
		},
	})
	return err
}

func (i *ingressInvocation) testServerPodSpec() apiv1.PodSpec {
	return apiv1.PodSpec{
		Containers: []apiv1.Container{
			{
				Name:  "server",
				Image: testServerImage,
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
	}
}

func (i *ingressInvocation) testServerServicePorts() []apiv1.ServicePort {
	return []apiv1.ServicePort{
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
	}
}

func (i *ingressInvocation) waitForTestServer() error {
	var err error
	var ep *apiv1.Endpoints
	for it := 0; it < maxRetryCount; it++ {
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

func getLoadBalancerURLs(provider string, k kubernetes.Interface, ing *api.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)
	var err error
	if provider == "minikube" {
		gomega.Eventually(func() error {
			var outputs []byte
			outputs, err = exec.Command(
				"/usr/local/bin/minikube",
				"service",
				ing.OffshootName(),
				"--url",
				"-n",
				ing.Namespace,
			).CombinedOutput()
			if err == nil {
				log.Infoln("Minikube exec output\n", string(outputs))
				for _, output := range strings.Split(string(outputs), "\n") {
					if strings.HasPrefix(output, "http") {
						serverAddr = append(serverAddr, output)
					}
				}
				return nil
			}
			return err
		}, "10m", "10s").Should(gomega.BeNil())
		if err != nil {
			return nil, err
		}
	} else {
		var svc *apiv1.Service
		gomega.Eventually(func() error {
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
							var doc bytes.Buffer
							err = defaultUrlTemplate.Execute(&doc, struct {
								IP   string
								Port int32
							}{
								ip,
								port,
							})
							if err != nil {
								return err
							}

							u, err := url.Parse(doc.String())
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
		}, "10m", "10s").Should(gomega.BeNil())
		if err != nil {
			return nil, err
		}
	}
	return serverAddr, nil
}

func getHostPortURLs(provider string, k kubernetes.Interface, ing *api.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)
	nodes, err := k.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(ing.NodeSelector()).String(),
	})
	if err != nil {
		return nil, err
	}

	var svc *apiv1.Service
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
			if addr.Type == apiv1.NodeExternalIP {
				for _, port := range ports {
					var doc bytes.Buffer
					err = defaultUrlTemplate.Execute(&doc, struct {
						IP   string
						Port int32
					}{
						addr.Address,
						port,
					})
					if err != nil {
						return nil, err
					}

					u, err := url.Parse(doc.String())
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

func (i *ingressInvocation) CheckTestServersPortAssignments(ing *api.Ingress) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	rc, err := i.KubeClient.CoreV1().ReplicationControllers(i.Config.TestNamespace).Get(i.TestServerName(), metav1.GetOptions{})
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

	rc.Spec.Replicas = types.Int32P(2)
	i.KubeClient.CoreV1().ReplicationControllers(rc.Namespace).Update(rc)
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
	return i.Config.CloudProviderName == "aws" ||
		i.Config.CloudProviderName == "gce" ||
		i.Config.CloudProviderName == "gke" ||
		i.Config.CloudProviderName == "azure" ||
		i.Config.CloudProviderName == "acs"
}
