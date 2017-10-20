package e2e

import (
	"time"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
		crt, key, err := f.CertManager.NewServerCertPair()
		Expect(err).NotTo(HaveOccurred())
		secret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Type: apiv1.SecretTypeTLS,
			Data: map[string][]byte{
				apiv1.TLSCertKey:       crt,
				apiv1.TLSPrivateKeyKey: key,
			},
		}
		_, err = f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		Expect(err).NotTo(HaveOccurred())
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
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create TCP", func() {
		BeforeEach(func() {
			ing.Spec.TLS = []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: secret.Name,
					},
					Hosts: []string{"http.appscode.test"},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4001),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
				{
					Host: "http.appscode.test",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4002),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
				{
					Host: "http.appscode.test",
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4003),
							NoTLS: true,
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
			Expect(svc.Spec.Ports[0].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))
			Expect(svc.Spec.Ports[1].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))
			Expect(svc.Spec.Ports[2].Port).To(Or(Equal(int32(4001)), Equal(int32(4002)), Equal(int32(4003))))

			var tcpNoSSL, tcpSSL, tcpWithNoSSL apiv1.ServicePort
			for _, p := range svc.Spec.Ports {
				if p.Port == 4001 {
					tcpNoSSL = p
				}

				if p.Port == 4002 {
					tcpSSL = p
				}

				if p.Port == 4003 {
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

	Describe("With Whitelist Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.WhitelistSourceRange] = "192.168.99.100"
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4001),
							NoTLS: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:  intstr.FromInt(4002),
							NoTLS: true,
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
			}
		})

		It("Should Add Whitelisted Ips to TCP Frontend", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			// Manually check if whitelisted ips are added to each tcp frontend rule of generated HAProxy config
			// TODO @ dipta: how to test if whitelist is actually working?
		})
	})

	Describe("Create TCP With Limit RPM", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromInt(4001),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromInt(4343),
							},
						},
					},
				},
			}
			ing.Annotations[api.LimitRPM] = "2"
		})

		It("Should test TCP Connections", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(4001)))

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoTCP(1, ing, eps, func(r *testserverclient.Response) bool {
				return false
			})
			Expect(err).To(HaveOccurred())

			// Wait for the rates to be reset
			time.Sleep(time.Minute * 2)

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *testserverclient.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4343"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
