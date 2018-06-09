package e2e

import (
	"io/ioutil"
	"net/http"

	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CertificateWithHTTPProvider", func() {
	var (
		f          *framework.Invocation
		ing        *api.Ingress
		cert       *api.Certificate
		userSecret *core.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()

		if !options.TestCertificate {
			Skip("Certificate Test is not enabled")
		}

		userSecret = &core.Secret{
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
			Domains: []string{framework.TestDomain, "other-" + framework.TestDomain},
			ChallengeProvider: api.ChallengeProvider{
				HTTP: &api.HTTPChallengeProvider{
					Ingress: api.LocalTypedReference{
						APIVersion: ing.APISchema(),
						Kind:       api.ResourceKindIngress,
						Name:       ing.Name,
					},
				},
			},
			ACMEUserSecretName: userSecret.Name,
			Storage: api.CertificateStorage{
				Secret: &core.LocalObjectReference{},
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
		if options.Cleanup {
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

			if meta.PossiblyInCluster() {
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))
				err = f.Ingress.DoHTTP(framework.MaxRetry, framework.TestDomain, ing, eps, http.MethodGet, "/.well-known/acme-challenge/", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r).ShouldNot(BeNil()) &&
						Expect(len(r.Body)).ShouldNot(Equal(0))
				})
				Expect(err).NotTo(HaveOccurred())
			} else {
				req, err := http.NewRequest(http.MethodGet, "http://127.1.0.1:56791"+"/.well-known/acme-challenge/", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Host", framework.TestDomain)
				req.Host = framework.TestDomain

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
