package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/appscode/go/types"
	hello "github.com/appscode/hello-grpc/pkg/apis/hello/v1alpha1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = FDescribe("IngressGRPC", func() {
	var (
		f              *framework.Invocation
		ing            *api.Ingress
		grpcController *core.ReplicationController
		grpcService    *core.Service
		err            error
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()

		By("Creating gRPC sample server")
		grpcController, err = createGRPCController(f)
		Expect(err).NotTo(HaveOccurred())

		By("Creating gRPC sample service")
		grpcService, err = createGRPCService(f)
		Expect(err).NotTo(HaveOccurred())

		ing.Spec.Rules = []api.IngressRule{
			{
				Host: "*",
				IngressRuleValue: api.IngressRuleValue{
					HTTP: &api.HTTPIngressRuleValue{
						Port:  intstr.FromInt(3001),
						Proto: "h2",
						Paths: []api.HTTPIngressPath{
							{
								Path: "/",
								Backend: api.HTTPIngressBackend{
									IngressBackend: api.IngressBackend{
										Proto:       "h2",
										ServiceName: grpcService.Name,
										ServicePort: intstr.FromInt(80),
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
			result, err := doGRPC(eps[0], "", &hello.IntroRequest{Name: "Voyager"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Intro).Should(Equal("hello, Voyager!"))
		})
	})
})

func doGRPC(address, crtPath string, request *hello.IntroRequest) (*hello.IntroResponse, error) {
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")

	option := grpc.WithInsecure()
	if len(crtPath) > 0 {
		creds, err := credentials.NewClientTLSFromFile(crtPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate")
		}
		option = grpc.WithTransportCredentials(creds)
	}

	conn, err := grpc.Dial(address, option)
	if err != nil {
		return nil, fmt.Errorf("did not connect, %v", err)
	}
	defer conn.Close()

	client := hello.NewHelloServiceClient(conn)
	return client.Intro(context.Background(), request)
}

func createGRPCController(f *framework.Invocation) (*core.ReplicationController, error) {
	return f.KubeClient.CoreV1().ReplicationControllers(f.Namespace()).Create(&core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.UniqueName(),
			Namespace: f.Namespace(),
			Labels: map[string]string{
				"app": "hello-grpc-" + f.App(),
			},
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Selector: map[string]string{
				"app": "hello-grpc-" + f.App(),
			},
			Template: &core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "hello-grpc-" + f.App(),
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "grpc-server",
							Image: "appscode/hello-grpc:0.1.0",
							Args: []string{
								"run",
								"--v=3",
							},
							Ports: []core.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	})
}

func createGRPCService(f *framework.Invocation) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(f.Namespace()).Create(&core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.UniqueName(),
			Namespace: f.Namespace(),
			Labels: map[string]string{
				"app": "hello-grpc-" + f.App(),
			},
		},
		Spec: core.ServiceSpec{
			Type: core.ServiceTypeNodePort, // TODO ClusterIP
			Ports: []core.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "hello-grpc-" + f.App(),
			},
		},
	})
}
