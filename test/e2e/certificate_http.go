package e2e

import (
	"io/ioutil"
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CertificateWithHTTPProvider", func() {
	var (
		f          *framework.Invocation
		ing        *api.Ingress
		cert       *api.Certificate
		userSecret *v1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()

		if !f.Config.TestCertificate {
			Skip("Certificate Test is not enabled")
		}

		userSecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Certificate.UniqueName(),
				Namespace: f.Namespace(),
			},
			Data: map[string][]byte{
				api.ACMEUserEmail: []byte("sadlil@appscode.com"),
				api.ACMEServerURL: []byte(certificate.LetsEncryptStagingURL),
			},
		}

		_, err := f.KubeClient.CoreV1().Secrets(userSecret.Namespace).Create(userSecret)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)

		cert = f.Certificate.GetSkeleton()
		cert.Spec = api.CertificateSpec{
			Domains: []string{"http.appscode.test", "test.appscode.test"},
			ChallengeProvider: api.ChallengeProvider{
				HTTP: &api.HTTPChallengeProvider{
					Ingress: api.LocalTypedReference{
						APIVersion:      ing.APISchema(),
						Kind:            api.ResourceKindIngress,
						Name:            ing.Name,
						UID:             ing.UID,
						ResourceVersion: ing.ResourceVersion,
					},
				},
			},
			ACMEUserSecretName: userSecret.Name,
			Storage: api.CertificateStorage{
				Secret: &apiv1.LocalObjectReference{},
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

		By("Creating certificate with" + cert.Name)
		err = f.Certificate.Create(cert)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(userSecret.Namespace).Delete(userSecret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create", func() {
		It("Should check secret", func() {
			Eventually(func() bool {
				updatedIngress, err := f.Ingress.Get(ing)
				Expect(err).NotTo(HaveOccurred())
				for _, rule := range updatedIngress.Spec.Rules {
					if rule.HTTP != nil {
						for _, path := range rule.HTTP.Paths {
							if path.Path == "/.well-known/acme-challenge/" {
								return true
							}
						}
					}
				}
				return false
			}, "20m", "10s").Should(BeTrue())

			if f.Config.InCluster {
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))
				err = f.Ingress.DoHTTP(framework.MaxRetry, "http.appscode.test", ing, eps, http.MethodGet, "/.well-known/acme-challenge/", func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r).ShouldNot(BeNil()) &&
						Expect(len(r.Body)).ShouldNot(Equal(0))
				})
				Expect(err).NotTo(HaveOccurred())
			} else {
				req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:56791"+"/.well-known/acme-challenge/", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Host", "http.appscode.test")
				req.Host = "http.appscode.test"

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).ShouldNot(BeNil())
				Expect(resp.Body).ShouldNot(BeNil())

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(body)).ShouldNot(Equal(0))
			}
		})
	})
})
