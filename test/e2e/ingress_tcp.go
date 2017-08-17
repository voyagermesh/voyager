package e2e

import (
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var _ = Describe("IngressTCP", func() {
	var (
		f      *framework.Invocation
		ing    *api.Ingress
		secret *apiv1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		secret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Type: apiv1.SecretTypeTLS,
			StringData: map[string]string{
				"tls.key": fakeHTTPAppsCodeDevKey,
				"tls.crt": fakeHTTPAppsCodeDevCert,
			},
		}
		_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		By("Creating ingress with name " + ing.GetName())
		err := f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExists(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create TCP", func() {
		BeforeEach(func() {
			ing.Spec.TLS = []api.IngressTLS{
				{
					SecretName: secret.Name,
					Hosts:      []string{"http.appscode.dev"},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4242),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
				{
					Host: "http.appscode.dev",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4141),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
				{
					Host: "http.appscode.dev",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4949),
							NoSSL: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
			}
		})

		It("Should test TCP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(3))
			Expect(svc.Spec.Ports[0].Port).To(Or(Equal(int32(4242)), Equal(int32(4141)), Equal(int32(4949))))
			Expect(svc.Spec.Ports[1].Port).To(Or(Equal(int32(4242)), Equal(int32(4141)), Equal(int32(4949))))
			Expect(svc.Spec.Ports[2].Port).To(Or(Equal(int32(4242)), Equal(int32(4141)), Equal(int32(4949))))

			var tcpNoSSL, tcpSSL, tcpWithNoSSL apiv1.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == 4242 {
					tcpNoSSL = p
				}

				if p.Port == 4343 {
					tcpSSL = p
				}

				if p.Port == 4949 {
					tcpWithNoSSL = p
				}
			}

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, f.Ingress.FilterEndpointsForPort(eps, tcpNoSSL), func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, f.Ingress.FilterEndpointsForPort(eps, tcpWithNoSSL), func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCPWithSSL(framework.MaxRetry, "", ing, f.Ingress.FilterEndpointsForPort(eps, tcpSSL), func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
