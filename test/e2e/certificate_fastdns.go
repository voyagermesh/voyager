/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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

var _ = Describe("CertificateWithDNSProviderFastDNS", func() {
	var (
		f                *framework.Invocation
		cert             *api.Certificate
		userSecret       *core.Secret
		credentialSecret *core.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()

		skipTestIfFastdnsSecretNotProvided()
		if !options.TestCertificate {
			Skip("FastDNS Certificate Test is not enabled")
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

		fmt.Println("TEST_AKAMAI_DNS_DOMAINS", os.Getenv("TEST_AKAMAI_DNS_DOMAINS"))

		credentialSecret = &core.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cred-" + f.Certificate.UniqueName(),
				Namespace: f.Namespace(),
			},
			Data: map[string][]byte{
				"AKAMAI_HOST":          []byte(os.Getenv("TEST_AKAMAI_HOST")),
				"AKAMAI_CLIENT_TOKEN":  []byte(os.Getenv("TEST_AKAMAI_CLIENT_TOKEN")),
				"AKAMAI_CLIENT_SECRET": []byte(os.Getenv("TEST_AKAMAI_CLIENT_SECRET")),
				"AKAMAI_ACCESS_TOKEN":  []byte(os.Getenv("TEST_AKAMAI_ACCESS_TOKEN")),
			},
		}

		_, err := f.KubeClient.CoreV1().Secrets(credentialSecret.Namespace).Create(credentialSecret)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		cert = f.Certificate.GetSkeleton()
		cert.Spec = api.CertificateSpec{
			Domains: []string{os.Getenv("TEST_AKAMAI_DNS_DOMAINS")},
			ChallengeProvider: api.ChallengeProvider{
				DNS: &api.DNSChallengeProvider{
					Provider:             "fastdns",
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
		if options.Cleanup {
			f.KubeClient.CoreV1().Secrets(userSecret.Namespace).Delete(userSecret.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Secrets(credentialSecret.Namespace).Delete(credentialSecret.Name, &metav1.DeleteOptions{})
			f.Certificate.Delete(cert)
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

func skipTestIfFastdnsSecretNotProvided() {
	if len(os.Getenv("TEST_AKAMAI_DNS_DOMAINS")) == 0 ||
		len(os.Getenv("TEST_AKAMAI_HOST")) == 0 ||
		len(os.Getenv("TEST_AKAMAI_CLIENT_TOKEN")) == 0 ||
		len(os.Getenv("TEST_AKAMAI_CLIENT_SECRET")) == 0 ||
		len(os.Getenv("TEST_AKAMAI_ACCESS_TOKEN")) == 0 {
		Skip("Skipping Test, FastDNS Secret Not Provided")
	}
}
