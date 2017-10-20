package e2e

import (
	"time"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressUpdates", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
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
		}
	})

	Describe("Secret Changed", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: "http.appscode.test",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: f.Ingress.TestServerName(),
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

		It("Should update when secret changed", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))

			crt, key, err := f.CertManager.NewServerCertPair()
			Expect(err).NotTo(HaveOccurred())
			secret := &apiv1.Secret{
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

			tobeUpdated, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())
			tobeUpdated.Spec.TLS = []api.IngressTLS{{SecretName: secret.Name, Hosts: []string{tobeUpdated.Spec.Rules[0].Host}}}
			err = f.Ingress.Update(tobeUpdated)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(time.Second * 15)
			Eventually(func() error {
				var err error
				svc, err = f.Ingress.GetOffShootService(ing)
				return err
			}, "10m", "5s")
			svc, err = f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(2))
			Expect(svc.Spec.Ports[0].Port).Should(Or(Equal(int32(443)), Equal(int32(80))))
			Expect(svc.Spec.Ports[1].Port).Should(Or(Equal(int32(443)), Equal(int32(80))))
		})
	})
})
