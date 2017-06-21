package e2e

import (
	"bytes"
	"math/rand"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	api "github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (s *IngressTestSuit) getURLs(baseIngress *api.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)
	var err error
	if s.t.Config.ProviderName == "minikube" {
		for i := 0; i < maxRetries; i++ {
			var outputs []byte
			log.Infoln("Running Command", "`minikube", "service", baseIngress.OffshootName()+" --url`")
			outputs, err = exec.Command(
				"/usr/local/bin/minikube",
				"service",
				baseIngress.OffshootName(),
				"--url",
				"-n",
				baseIngress.Namespace,
			).CombinedOutput()
			if err == nil {
				log.Infoln("Output\n", string(outputs))
				for _, output := range strings.Split(string(outputs), "\n") {
					if strings.HasPrefix(output, "http") {
						serverAddr = append(serverAddr, output)
					}
				}
				return serverAddr, nil
			}
			log.Infoln("minikube service returned with", err, string(outputs))
			time.Sleep(time.Second * 10)
		}
		if err != nil {
			return nil, errors.New().WithCause(err).WithMessage("Failed to load service from minikube").Err()
		}
	} else {
		var svc *apiv1.Service
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
			if err == nil {
				if len(svc.Status.LoadBalancer.Ingress) != 0 {
					ips := make([]string, 0)
					for _, ingress := range svc.Status.LoadBalancer.Ingress {
						if s.t.Config.ProviderName == "aws" {
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
								return nil, errors.New().WithCause(err).Err()
							}

							u, err := url.Parse(doc.String())
							if err != nil {
								return nil, errors.New().WithCause(err).Err()
							}

							serverAddr = append(serverAddr, u.String())
						}
					}
					return serverAddr, nil
				}
			}
			time.Sleep(time.Second * 10)
			log.Infoln("Waiting for service to be created")
		}
		if err != nil {
			return nil, errors.New().WithCause(err).Err()
		}
	}
	return serverAddr, nil
}

func (s *IngressTestSuit) getDaemonURLs(baseIngress *api.Ingress) ([]string, error) {
	serverAddr := make([]string, 0)
	nodes, err := s.t.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(baseIngress.NodeSelector()).String(),
	})
	if err != nil {
		return nil, errors.New().WithCause(err).Err()
	}

	var svc *apiv1.Service
	var ports []int32
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			if len(svc.Spec.Ports) > 0 {
				for _, port := range svc.Spec.Ports {
					ports = append(ports, port.Port)
				}
			}
			break
		}
		time.Sleep(time.Second * 10)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return nil, errors.New().WithCause(err).Err()
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
						return nil, errors.New().WithCause(err).Err()
					}

					u, err := url.Parse(doc.String())
					if err != nil {
						return nil, errors.New().WithCause(err).Err()
					}
					serverAddr = append(serverAddr, u.String())
				}
			}
		}
	}
	return serverAddr, nil
}

func testIngressName() string {
	return "test-ings-" + randString(8)
}

var alphanums = []rune("abcdefghijklmnopqrstuvwxz0123456789")

func randString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = alphanums[rand.Intn(len(alphanums))]
	}
	return string(b)
}
