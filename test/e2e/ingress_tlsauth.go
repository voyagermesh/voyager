package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"time"
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
		tlsSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			StringData: map[string]string{
				"tls.crt": serverCertAppsCodeTest,
				"tls.key": serverKeyAppsCodeTest,
			},
			Type: apiv1.SecretTypeTLS,
		}
		_, err := f.KubeClient.CoreV1().Secrets(tlsSecret.Namespace).Create(tlsSecret)
		Expect(err).NotTo(HaveOccurred())

		caSecret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			StringData: map[string]string{
				"ca.crt": caCertAppsCodeTest,
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

			time.Sleep(time.Hour)

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "http.appscode.test", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				fmt.Println(*r)
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			fmt.Println("========================", err)
			// Expect(err).NotTo(HaveOccurred())

			// TLS Auth
			clientCert, err := tls.X509KeyPair([]byte(clientCertAppsCodeTest), []byte(clientKeyAppsCodeTest))
			Expect(err).NotTo(HaveOccurred())

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM([]byte(caCertAppsCodeTest))

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      caCertPool,
			}
			tlsConfig.BuildNameToCertificate()
			tr := &http.Transport{TLSClientConfig: tlsConfig}

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

const (
	caCertAppsCodeTest = `-----BEGIN CERTIFICATE-----
MIIF2jCCA8KgAwIBAgIJANV3irftFN4OMA0GCSqGSIb3DQEBCwUAMIGBMQswCQYD
VQQGEwJCRDEOMAwGA1UECAwFRGhha2ExDjAMBgNVBAcMBURoYWthMREwDwYDVQQK
DAhBcHBzQ29kZTEbMBkGA1UEAwwSaHR0cC5hcHBzY29kZS50ZXN0MSIwIAYJKoZI
hvcNAQkBFhNzYWRsaWxAYXBwc2NvZGUuY29tMB4XDTE3MTAxMTA5NDYxMVoXDTE4
MTAxMTA5NDYxMVowgYExCzAJBgNVBAYTAkJEMQ4wDAYDVQQIDAVEaGFrYTEOMAwG
A1UEBwwFRGhha2ExETAPBgNVBAoMCEFwcHNDb2RlMRswGQYDVQQDDBJodHRwLmFw
cHNjb2RlLnRlc3QxIjAgBgkqhkiG9w0BCQEWE3NhZGxpbEBhcHBzY29kZS5jb20w
ggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDVE22WPWZt8oxeUQH794C+
BaUgd6VrpVqhEMR2rRb8VuzLWfG/FZ9dL/cbPGqja7JcgxXsJKtX0m+SA+VHZtPe
zqEDm5GcDiJbPsfKgKrZuVxjjUlLgimoCFJsOxITXIY2DCCtbycK5lY7JbR62piJ
StPLbmO8LlMiHH1yejumhapZDQ2U2tnmeeFlro/35iSRUS5MgkWTHk9nRTRIUoko
gFywZToBePn8q3mC4N+XOteAtBdxcil8V3MMv55CgNg8nWp+jgRpvFFGf/l13xfJ
dpL/7O5CwxMaPedQb/1Q5Ulc5tqeYa2GFMIIuzYNchXm1mhk20t/2YkjjGI6N5oZ
B42ISCLrF7nSlMchTjH/aJCeAosqP6O3QMZ1eRhMmTxZpWKk9CPHfZ3OBB5rpcGI
xvtm5mUKSGJ9KToUDlm6GY3AOIgU6falFkyXvIX/SVHxZNOFq8rq8FudNnJWlki5
5+xrn21l6Xuw45y27LjNai3xqdUWY3GfrA3oIrFkEdlDwFng+ggDgbkpln3qk/c2
bTrr7AUswnhS4BouxNSxkwNfDYm0ijkT+VYZBIO6JbILa+OY0mgTYME6VX/vz+SB
bOM+k8PrNMhIyM9WTvcQlrf39XxqONzdDsDj6bUfN0Dvt9dNGrbYxw1rLKfv0I5L
uCAVubqYXqFV7BOAeLG3QwIDAQABo1MwUTAdBgNVHQ4EFgQUB/I6OSU8/WLIMR9x
RvHd0BkKi3gwHwYDVR0jBBgwFoAUB/I6OSU8/WLIMR9xRvHd0BkKi3gwDwYDVR0T
AQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAgEAt+p6mmDoI3jusWwRcYuYx7kJ
W5k8NQYNxEYRy/XHeNcbwo+qkdhNdRyj+Py2kLc0JCJ/A+lSl3q35Qw5sT1rf32w
3vINeFxPxkDYRb2QJvFhxu54GoK8AXGxahTT2ogdlXNhYDFEEGrE+csDYjJiSOCm
0jClexWinFaRHC8H9j50uQmlRZkcT1dovjlZ3ztR+ncN86pPuDrQWuXCsrJpB5kU
CnWf4YTDgj/PsyRj//qRJt6OzAk+zRZhlwlHbSukt++EKkNGJbBzUiERh+DkQENz
oxV69idbpauIiLPLbgNxA+5Qb6pnPDyKz0ECTZ1UUGlrgilCNAxeylzer4Ra9hE9
ucHVyjk76jhnmg1gl+L+KyZgqchumUAFIGDxnlOby4+w4q0g3j/j5YgbV5gEfr8M
8Fwn9HL2X7T0dJErRc8gWWf6J4z6ECELzvoKY2txD/OGF0nKoHN6eluOI+i4yM4P
OelZzMAMcBkNii2fnyDlRshKgt9CEJnWZazZqpz7hRfZVwJ3rTzkW6h+tGSV9LNR
Dn2cPt+L7t2nW8O0lV3GSVE/Mu+Emwdwy1YJI7ggDY45is+77kuaOE8Mwrx8Yxp/
a7Qzm3rkvNPAHe/nhIE0FFj3c8iKtNB+h2YRS4Mizx0//LBR+4vF+XLY6nHeopzY
H4+lUdxQiEIN7MuVUIo=
-----END CERTIFICATE-----
`
	serverCertAppsCodeTest = `-----BEGIN CERTIFICATE-----
MIID9DCCAdwCAQEwDQYJKoZIhvcNAQELBQAwgYExCzAJBgNVBAYTAkJEMQ4wDAYD
VQQIDAVEaGFrYTEOMAwGA1UEBwwFRGhha2ExETAPBgNVBAoMCEFwcHNDb2RlMRsw
GQYDVQQDDBJodHRwLmFwcHNjb2RlLnRlc3QxIjAgBgkqhkiG9w0BCQEWE3NhZGxp
bEBhcHBzY29kZS5jb20wHhcNMTcxMDExMDk0ODE4WhcNMTgxMDExMDk0ODE4WjCB
gTELMAkGA1UEBhMCQkQxDjAMBgNVBAgMBURoYWthMQ4wDAYDVQQHDAVEaGFrYTER
MA8GA1UECgwIQXBwc0NvZGUxGzAZBgNVBAMMEmh0dHAuYXBwc2NvZGUudGVzdDEi
MCAGCSqGSIb3DQEJARYTc2FkbGlsQGFwcHNjb2RlLmNvbTCBnzANBgkqhkiG9w0B
AQEFAAOBjQAwgYkCgYEAvEZZRf0yrsVjq7XWZd6wK8efjUyGZmx8dW4q/dTIePDK
6nSdzc8ZoYJEzsZporodVoktjkszDHGtQEvHevFuWDi0R4fmrJ872qCh95LscRK/
Z9cF50H4D5q4NYwn/kga7ndY8o+4WEUMJ+YhTxS7XPfp5B/yA6GJIYb9GAZtj0EC
AwEAATANBgkqhkiG9w0BAQsFAAOCAgEAEU8VOI+mjIgYwiUhKiRPNQaI1gx+ciLt
cRd9wqJVDUkt/k4SrGjXSXFcsSN9SPY+dna1Lan37YmFAld6Q80NhKK6nTDr0YO0
qIu+T9+yCyMkBazXA16s4lTJ/xs1wvCVT1W8vaTwL8j5j9rSqJh6isZhX2rqg1oz
sSKVid13TSCPtRz8oF4EkRwzIc/FUgQ3mg6wOMJic6GyFPPldD0aGpNE9/8mwxxX
gsAjAdZ02GMyRHfA37JX06yvPGKzZgu4hgDZaW/O82d6j6CrPmEvSFFmqLZo0i/c
mtY2i/4kJnccyDaZRdW4Iy1I9J+HWBzoQ/GqHTkbC6BNGXuztAwetx9e/nV0n0Gr
D8XvXa7qnFxOKwpv50jHzrge5rv2sidGCSdkdxz3WSH+NvBXxrl0q5FRnrcXekPf
UXHoFYueJAufMKtqZuIvD+eoOYSxsQF8A9C2S85giTotOt+FvGBA/a6sQQYQWhGu
cQrWvLtWcwDCqxYrkyGHQct6A/A28L5R9ChFsJDARfgO/gPlZPy0TvevQ7X0wd8O
ndxJSc5PrDXktZAlsKWL78+VMmLc47lgCTvxI3TvGuyR4mc9XoDjeRi5APhXoGvv
XtI4LyiRovvxx47Im4y5GZ1FVHkRD0NnWb/8vRQKcYFmTXN2XM53GOs55BoErOGy
1ceHPjFf5H0=
-----END CERTIFICATE-----
`
	serverKeyAppsCodeTest = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQC8RllF/TKuxWOrtdZl3rArx5+NTIZmbHx1bir91Mh48MrqdJ3N
zxmhgkTOxmmiuh1WiS2OSzMMca1AS8d68W5YOLRHh+asnzvaoKH3kuxxEr9n1wXn
QfgPmrg1jCf+SBrud1jyj7hYRQwn5iFPFLtc9+nkH/IDoYkhhv0YBm2PQQIDAQAB
AoGAAZ7JXeTT7tUcCnpCIhZMhdPt95VVAsSkPY55KD4Qi5nm5SWjxgWmPtGULgNf
HVtkrT34+mSDR/QifY8pZFF3lZ7+V9lc/nyp4dwt5TMagRfarKhXcXT5psLPTO/b
TaEFYdJoJK74Th5DY2hj24VbsPPhUaWu7gbhaGGXh34uAxECQQDokqzsm1GAQjMt
W7XVhQcfn+Dm+uTPWpZQcrz6mV3u74OEtywFypWGzHRuRsIZ3oh518PXVsoKfFac
stpksTcDAkEAzz1dAmaocIJ+kkAvIWWCTDFzaaSE+zCoSVXtJlPaTOjEAhSSSe8V
GFhca3pdOPNiTdWEmDshKl7UcoA3aMTbawJAa/k8oxRwwBT74YEEaD68UehN565v
b/zkcDD0n3t4aqdz7beEjqPAy7Y8D751+sCfp8GOQHkgj8GuDE3Uqo7DtQJAVvdb
Rpyp5rz8PIduv8cHTM4brqN5oqeil1PVtxXNMCUly/GyChCoe5TpA7lP0YxhHmSR
xghaPJua74natr6VjQJBAMfdhod8J/GprULDMsrjqc31v6JjGScdgGR5A0SbGIFT
nM2L+NpsGu8CJk+ioulyqKiUqLfHubzmLStZbyxlRdc=
-----END RSA PRIVATE KEY-----
`
	clientCertAppsCodeTest = `-----BEGIN CERTIFICATE-----
MIID9DCCAdwCAQEwDQYJKoZIhvcNAQELBQAwgYExCzAJBgNVBAYTAkJEMQ4wDAYD
VQQIDAVEaGFrYTEOMAwGA1UEBwwFRGhha2ExETAPBgNVBAoMCEFwcHNDb2RlMRsw
GQYDVQQDDBJodHRwLmFwcHNjb2RlLnRlc3QxIjAgBgkqhkiG9w0BCQEWE3NhZGxp
bEBhcHBzY29kZS5jb20wHhcNMTcxMDExMDk0OTE2WhcNMTgxMDExMDk0OTE2WjCB
gTELMAkGA1UEBhMCQkQxDjAMBgNVBAgMBURoYWthMQ4wDAYDVQQHDAVEaGFrYTER
MA8GA1UECgwIQXBwc0NvZGUxGzAZBgNVBAMMEmh0dHAuYXBwc2NvZGUudGVzdDEi
MCAGCSqGSIb3DQEJARYTc2FkbGlsQGFwcHNjb2RlLmNvbTCBnzANBgkqhkiG9w0B
AQEFAAOBjQAwgYkCgYEA7sSSuZNngd/GcFTeg3CIKWRnXXKjXu2FKJmf6TylH83w
hJ8yYNYQCbqC5/H8zCXKVjg1ZOuVPZuqquKOB4qc2+Z8J/ls5IPcb2t5k7pNk7ao
9e6F7eD8DVqOQp//vNxK4qhRuQUk10vJ9/0M+9f6JNTGRFC6tdpgQVr3iZb+g4sC
AwEAATANBgkqhkiG9w0BAQsFAAOCAgEACrQcI7cYiYgAPlhxfBI1TAZqYoQ7ckGn
9o2XBwlWo4hiA4SHUjpxFIZYr34+ByLdZ3k5FSh2P02WMK3uVSOmtv3pvH/tkZoW
Q2dBjKYNdPUwoYjPfjZ4uLJAlNwO+SMRjXOuwX5pFRs7XSRWo1Neu9NsIiNx+wnr
zHOm9xkK28iIgjJ7bh/Wl0UTV1ihoEjBXaNc2v/9dsbEraizOG5o5jMSyP//saYY
FCqMX+csOEOn6Pb9g4c81n+Ea6M9fVON6Fa7wu9Twzw2RyFYQDSuEzwM/GA55mXk
6Q73IHysMa1+JX9OjWu1pFkg0AzKTWNxxWYxSA0RB9CohM/x7PqZ77eFGeZasWGi
kKtWKnR3Sixz1s0J7cE9xgpZgprhTGyy2CjAIeZIlOQRJsSLMszRmoQ0eVYG7XR8
vLKqBsuLRH50bO8k+zFYc3RQXhuBexEsCWlOUsXjoD76ZUkmwGcHYN6uLEi+OviQ
jjk8cqLdX/wOohcE9zViFZBht/taW+3cybesG6R6iyNMXOioxpho6NTkw7xlxy8S
S3iInqBIyjK1k0Wknr0sq6SUC5xSkG8Pe/61bDmfXSme4PRadtTbO4/k/yumRp+s
FLxqKXFUIZvjB4xPr1JjtEI0wAUgZtg9eREFOYqN03OWDilkufKcHE3u1RTxm9B+
/SR+h0MNnbE=
-----END CERTIFICATE-----
`
	clientKeyAppsCodeTest = `-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDuxJK5k2eB38ZwVN6DcIgpZGddcqNe7YUomZ/pPKUfzfCEnzJg
1hAJuoLn8fzMJcpWODVk65U9m6qq4o4Hipzb5nwn+Wzkg9xva3mTuk2Ttqj17oXt
4PwNWo5Cn/+83EriqFG5BSTXS8n3/Qz71/ok1MZEULq12mBBWveJlv6DiwIDAQAB
AoGBAKdbQTyyBSsTHpQ96HlYtxfMOGdXows2kM8UXvGsgFD6mEtdCoK1iChJgtfw
1bCCDIDChSpntgOoyMdeZQ8EKUzeX0S0/+JZu/5jvzu+ZxEMZXNqA+41zAiACk/a
mzOCKk5gip1FZKSp5PvwUi/R0TC/9xrk2eTl5H/zNN6CczQBAkEA/XZxmTeF6ZGk
EL26AJdkR+/pVKxLJdsCRrJtPdSINAXRXIfscV76IFNP0Tbi6ciEyJKKF+xH75hh
oLKu3A1B4QJBAPEoeHhCFFyY8vI+IImie3kehd8uejHX2KFw7BgCSc3/9a7NpUEa
mzGmxl6VRZTA0F66OIFzg2hdjb1c64rISusCQQCqVsuJePMaQbLNPXSfqR7P6cAa
E6B9VG53LLqV7xuKOs61LPQOTRI0X0kpBYYCL6xtT25XHYhK0VHrOaqiYJaBAkEA
zvGqx4/1By0dNjGIHHP5PwupV8b7hzAxrwBHKac1DHi8rL++MusRCH+UNPAloKwB
Y3isKrIkrvexPTGy0wpj9wJAJRvQV1udMrguWna3qFp4sWi89VbC1wkLIbAVMKcz
jH0fUjyA4BDjqs7Ng0WQyRPXmzvILjgKgHjiflNqZV6gcw==
-----END RSA PRIVATE KEY-----
`
)
