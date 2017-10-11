package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"io/ioutil"
	"os"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var _ = Describe("IngressWithTLSAuth", func() {
	var (
		f                   *framework.Invocation
		ing                 *api.Ingress
		tlsSecret, caSecret *apiv1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		crt, key, err := f.CertManager.NewServerCertPair()
		Expect(err).NotTo(HaveOccurred())
		tlsSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Data: map[string][]byte{
				apiv1.TLSCertKey:       crt,
				apiv1.TLSPrivateKeyKey: key,
			},
			Type: apiv1.SecretTypeTLS,
		}
		_, err = f.KubeClient.CoreV1().Secrets(tlsSecret.Namespace).Create(tlsSecret)
		Expect(err).NotTo(HaveOccurred())

		caSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Data: map[string][]byte{
				"ca.crt": f.CertManager.CACert(),
			},
		}
		_, err = f.KubeClient.CoreV1().Secrets(caSecret.Namespace).Create(caSecret)
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
			f.KubeClient.CoreV1().Secrets(tlsSecret.Namespace).Delete(tlsSecret.Name, &metav1.DeleteOptions{})
			f.KubeClient.CoreV1().Secrets(caSecret.Namespace).Delete(caSecret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			if f.Config.CloudProviderName == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}
			ing.Spec = api.IngressSpec{
				FrontendRules: []api.FrontendRule{
					{
						Port: intstr.FromInt(443),
						Auth: &api.AuthOption{
							TLS: &api.TLSAuth{
								SecretName:   caSecret.Name,
								VerifyClient: api.TLSAuthVerifyRequired,
								ErrorPage:    "https://http.appscode.test/testpath/ok",
							},
						},
					},
				},
				TLS: []api.IngressTLS{
					{
						Ref: &api.LocalTypedReference{
							Kind: "Secret",
							Name: tlsSecret.Name,
						},
						Hosts: []string{"http.appscode.test"},
					},
				},
				Rules: []api.IngressRule{
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
				},
			}
		})

		FIt("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			ccrt, ckey, err := f.CertManager.NewClientCertPair()
			Expect(err).NotTo(HaveOccurred())
			clientCert, err := tls.X509KeyPair(ccrt, ckey)
			Expect(err).NotTo(HaveOccurred())

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(f.CertManager.CACert())

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      caCertPool,
			}
			tlsConfig.BuildNameToCertificate()
			tr := &http.Transport{TLSClientConfig: tlsConfig}

			if len(f.Config.DumpLocation) > 0 {
				ioutil.WriteFile(f.Config.DumpLocation+"/ca.crt", f.CertManager.CACert(), os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.crt", ccrt, os.ModePerm)
				ioutil.WriteFile(f.Config.DumpLocation+"/client.key", ckey, os.ModePerm)
			}

			//time.Sleep(time.Hour)

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "http.appscode.test", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				fmt.Println(*r)
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			fmt.Println("========================", err)
			// Expect(err).NotTo(HaveOccurred())

			// TLS Auth
			err = f.Ingress.DoHTTPsWithTransport(framework.MaxRetry, "http.appscode.test", tr, ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				fmt.Println(*r)
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath"))
			})
			Expect(err).NotTo(HaveOccurred())

		})
	})
})
