package e2e

import (
	"fmt"
	"os"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CertificateWithDNSProvider", func() {
	var (
		f                *framework.Invocation
		cert             *api.Certificate
		userSecret       *core.Secret
		credentialSecret *core.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()

		skipTestIfSecretNotProvided()
		if !f.Config.TestCertificate {
			Skip("Certificate Test is not enabled")
		}

		userSecret = &core.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "user-" + f.Certificate.UniqueName(),
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
		f = root.Invoke()

		fmt.Println("TEST_GCE_PROJECT", os.Getenv("TEST_GCE_PROJECT"))
		fmt.Println("TEST_DNS_DOMAINS", os.Getenv("TEST_DNS_DOMAINS"))

		credentialSecret = &core.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cred-" + f.Certificate.UniqueName(),
				Namespace: f.Namespace(),
			},
			Data: map[string][]byte{
				"GCE_PROJECT":              []byte(os.Getenv("TEST_GCE_PROJECT")),
				"GCE_SERVICE_ACCOUNT_DATA": []byte(os.Getenv("TEST_GCE_SERVICE_ACCOUNT_DATA")),
			},
		}

		_, err := f.KubeClient.CoreV1().Secrets(credentialSecret.Namespace).Create(credentialSecret)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		cert = f.Certificate.GetSkeleton()
		cert.Spec = api.CertificateSpec{
			Domains: []string{os.Getenv("TEST_DNS_DOMAINS")},
			ChallengeProvider: api.ChallengeProvider{
				DNS: &api.DNSChallengeProvider{
					Provider:             "googlecloud",
					CredentialSecretName: credentialSecret.Name,
				},
			},
			ACMEUserSecretName: userSecret.Name,
			Storage: api.CertificateStorage{
				Secret: &core.LocalObjectReference{},
			},
		}
	})

	JustBeforeEach(func() {
		By("Creating certificate with" + cert.Name)
		err := f.Certificate.Create(cert)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.KubeClient.CoreV1().Secrets(userSecret.Namespace).Delete(userSecret.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Secrets(credentialSecret.Namespace).Delete(credentialSecret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create", func() {
		It("Should check secret", func() {
			Eventually(func() bool {
				secret, err := f.KubeClient.CoreV1().Secrets(cert.Namespace).Get(cert.SecretName(), metav1.GetOptions{})
				if err != nil {
					return false
				}
				if _, ok := secret.Data["tls.crt"]; !ok {
					return false
				}
				return true
			}, "20m", "10s").Should(BeTrue())
		})
	})
})

func skipTestIfSecretNotProvided() {
	if len(os.Getenv("TEST_GCE_PROJECT")) == 0 ||
		len(os.Getenv("TEST_GCE_SERVICE_ACCOUNT_DATA")) == 0 {
		Skip("Skipping Test, Secret Not Provided")
	}
}
