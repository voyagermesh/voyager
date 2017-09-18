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
)

var _ = Describe("CertificateWithHTTPProvider", func() {
	var (
		f    *framework.Invocation
		ing  *api.Ingress
		cert *api.Certificate
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)

		cert = f.Certificate.GetSkeleton()
		cert.Spec = api.CertificateSpec{
			Provider:                     "http",
			Domains:                      []string{"http.appscode.dev", "test.appscode.dev"},
			Email:                        "sadlil@appscode.com",
			ACMEServerURL:                certificate.LetsEncryptStagingURL,
			HTTPProviderIngressReference: *ing.ObjectReference(),
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
				err = f.Ingress.DoHTTP(framework.MaxRetry, "http.appscode.dev", ing, eps, http.MethodGet, "/.well-known/acme-challenge/", func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r).ShouldNot(BeNil()) &&
						Expect(len(r.Body)).ShouldNot(Equal(0))
				})
				Expect(err).NotTo(HaveOccurred())
			} else {
				req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:56791"+"/.well-known/acme-challenge/", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Host", "http.appscode.dev")
				req.Host = "http.appscode.dev"

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
