package e2e

import (
	"fmt"

	"github.com/appscode/go/types"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressGRPC", func() {
	var (
		f                                  *framework.Invocation
		ing                                *api.Ingress
		tlsSecretHAProxy, tlsSecretBackend *core.Secret
		grpcController                     *core.ReplicationController
		grpcService                        *core.Service
		err                                error
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()

		By("Creating TLS secrets for HAProxy")
		tlsSecretHAProxy, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{framework.TestDomain})
		Expect(err).NotTo(HaveOccurred())

		By("Creating TLS secrets for Backend Server")
		tlsSecretBackend, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{framework.TestDomain})
		Expect(err).NotTo(HaveOccurred())

		By("Creating gRPC sample server")
		grpcController, err = createGRPCController(f, tlsSecretBackend.Name)
		Expect(err).NotTo(HaveOccurred())

		By("Creating gRPC sample service")
		grpcService, err = createGRPCService(f, "/tmp/server-crt/tls.crt")
		Expect(err).NotTo(HaveOccurred())

		ing.Spec.TLS = []api.IngressTLS{
			{
				Hosts:      []string{"*"},
				SecretName: tlsSecretHAProxy.Name,
			},
		}
		// TODO
		/*ing.Spec.ConfigVolumes = []api.VolumeSource{
			{
				Name:      "server-crt-volume",
				MountPath: "/tmp/server-crt",
				Secret: &core.SecretVolumeSource{
					SecretName: tlsSecretBackend.Name,
				},
			},
		}*/
		ing.Spec.Rules = []api.IngressRule{
			{
				Host: "*",
				IngressRuleValue: api.IngressRuleValue{
					HTTP: &api.HTTPIngressRuleValue{
						Port: intstr.FromInt(3001),
						ALPN: []string{"h2", "http/1.1"},
						Paths: []api.HTTPIngressPath{
							{
								Path: "/",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										ALPN:        []string{"h2", "http/1.1"},
										ServiceName: grpcService.Name,
										ServicePort: intstr.FromInt(3000), // gRPC port
									},
								},
							},
						},
					},
				},
			},
		}
	})

	JustBeforeEach(func() {
		By("Creating ingress with name " + ing.GetName())
		err := f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if options.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(tlsSecretHAProxy.Namespace).Delete(tlsSecretHAProxy.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Secrets(tlsSecretBackend.Namespace).Delete(tlsSecretBackend.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Services(f.Ingress.Namespace()).Delete(grpcService.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().ReplicationControllers(f.Ingress.Namespace()).Delete(grpcController.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("gRPC Request", func() {
		It("Should get response from gRPC server", func() {
			By("Getting gRPC endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1)) // 3001

			By("Requesting gRPC in endpoint " + eps[0])
			err = doGRPC(eps[0])
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func doGRPC(url string) error {
	// TODO
	return nil
}

func createGRPCController(f *framework.Invocation, tlsSecretName string) (*core.ReplicationController, error) {
	return f.KubeClient.CoreV1().ReplicationControllers(f.Namespace()).Create(&core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.UniqueName(),
			Namespace: f.Namespace(),
			Labels: map[string]string{
				"app": "test-grpc-server-" + f.App(),
			},
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Selector: map[string]string{
				"app": "test-grpc-server-" + f.App(),
			},
			Template: &core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-grpc-server-" + f.App(),
					},
				},
				Spec: core.PodSpec{
					Volumes: []core.Volume{
						{
							Name: "tls-secret",
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: tlsSecretName,
								},
							},
						},
					},
					Containers: []core.Container{
						{
							Name:  "grpc-server",
							Image: "diptadas/sample-grpc-server:1.0",
							Args: []string{
								"--crt=/tmp/server-crt/tls.crt",
								"--key=/tmp/server-crt/tls.key",
							},
							Ports: []core.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 3000,
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "tls-secret",
									MountPath: "/tmp/server-crt",
								},
							},
						},
					},
				},
			},
		},
	})
}

func createGRPCService(f *framework.Invocation, caFilePath string) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(f.Namespace()).Create(&core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.UniqueName(),
			Namespace: f.Namespace(),
			Labels: map[string]string{
				"app": "test-grpc-server-" + f.App(),
			},
			Annotations: map[string]string{
				"ingress.appscode.com/backend-tls": fmt.Sprintf("ssl ca-file %s", caFilePath),
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "grpc",
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "test-grpc-server-" + f.App(),
			},
		},
	})
}
