package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

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
			ing.Spec = api.IngressSpec{
				FrontendRules: []api.FrontendRule{
					{
						Port: intstr.FromInt(443),
						Auth: &api.AuthOption{
							TLS: &api.TLSAuth{
								SecretName:   tlsSecret.Name,
								VerifyClient: api.TLSAuthVerifyRequired,
								ErrorPage:    "https://http.appscode.test/testpath/ok",
							},
						},
					},
				},
				TLS: []api.IngressTLS{
					{
						SecretName: tlsSecret.Name,
						Hosts:      []string{"http.appscode.dev"},
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

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPsTestRedirect(framework.MaxRetry, "http.appscode.test", ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(301)) &&
					Expect(r.ResponseHeader).Should(HaveKey("Location")) &&
					Expect(r.ResponseHeader.Get("Location")).Should(Equal("https://http.appscode.test/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

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
MIIFojCCA4qgAwIBAgIJAORbwSMs0JgOMA0GCSqGSIb3DQEBCwUAMGYxCzAJBgNV
BAYTAkJEMRMwEQYDVQQIDApTb21lLVN0YXRlMQ4wDAYDVQQHDAVEaGFrYTEVMBMG
A1UECgwMYXBwc2NvZGUuY29tMRswGQYDVQQDDBJodHRwLmFwcHNjb2RlLnRlc3Qw
HhcNMTcxMDEwMDY1MjIyWhcNMTgxMDEwMDY1MjIyWjBmMQswCQYDVQQGEwJCRDET
MBEGA1UECAwKU29tZS1TdGF0ZTEOMAwGA1UEBwwFRGhha2ExFTATBgNVBAoMDGFw
cHNjb2RlLmNvbTEbMBkGA1UEAwwSaHR0cC5hcHBzY29kZS50ZXN0MIICIjANBgkq
hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA1LBM7CaHDxOvsRWb4uiMXVx1feEvJiJf
34LexH7BBRULUw1wAvFVnOUvpRjel2c0rpi2hLu8NgU+SqX6RB86KtbLYq5CLAqz
UBhdgVqYdkSPmswZrsQs+UT6AdGgXOZTzecpHpEK4wLkpLGELGtt09E83Trwx/xM
a/A0XmmJHlKQidLBwtVnDPPAkQL7f3DJruw8fmLyOO7kshk9gxE37MCAu80HKnh+
3D7IAXUhngadB/tqBwMrj2rfDxRs5mjo8HMrfSuNiFPe6+LNXILG5A2ClANHh126
lP+e/Uqo25Bh58rnEsmXaec/2UHO7v6BZMqYLKZVMTrYgM0GA8K56GU4jYVO4XDB
iRV+XCWewR5ocES+ed4UWKltqFh07B1tT5CtLmWHmFUYX4cU6AMje2AAPmMlR6ik
T5727vdBJ67+6u1uSiJsfyr8CH5zG/0+poJGuHYqc0EKpKtbIwF8L3irsC87A/3C
h7Ty2XBQfIhAhwqlh4vfC2Bd/s5ClRPQyeDIqrSgTcA4gL6mianR4w/cxvw+mgoA
hmy3XnQhthb/jzXb6ERtbfuPqAVFGPW4RNQ0ExReNMBtjueM8Lm91GxJFpf+dnwp
iNFuPwaJciUYXxp66tbm+plRP9C1uPy75gPs4jjZqr6udjBQme8ynUWSL3YxtTCD
aLkI+c5CsXkCAwEAAaNTMFEwHQYDVR0OBBYEFPyv4okdxJELXg99PiY9KWCsmvSG
MB8GA1UdIwQYMBaAFPyv4okdxJELXg99PiY9KWCsmvSGMA8GA1UdEwEB/wQFMAMB
Af8wDQYJKoZIhvcNAQELBQADggIBAGYD65h3WhYCZZwtj8hUm2hXv7d2yOMRcUol
Bn4FxvBEeRg4g9XKrNVysjUm8OBud289WcdtE780eDK4wkDMq7yeLMKEGyDDab9G
etg/pZk+RrV5n2hyaAZwvPkiMSIaEM0KAJlhOpre9KPM4+3Xc0xPqN6cy41K8f0s
jmc2wQTage1BSsIQxIRlJnuY74D3g1HgHSN/s5N9WxAgTXI62YtFF4t5AxfGhxbV
Xkv6mqw2x2qVrg+ISJs+xifECD+pcsPx0FDDTumOrWY6AI8w3xHf2KLgnCD0UhIE
rg0nHMWKY6MJR2RXIEDZbnpPtOa9C6FsR9zkMkAi0YrVQUP/owLiUTeYjx8OmD0T
ypxfN0GyOwifq0EwjM/aSrzscU6LG+0RehiRTqcDTDJol5G/+nrZib8Mysu4HsOI
5S9Dh8Yx1KLt3CXODtGYermLJ3tpTsLuOgAEDph8+V470NuAq6VVCiAv8k8Ou7C0
sNWtQyRUawFUBGi8P6n3Mw7Jdy8SvmdTLx+FounsMq80pH1w201cf/H9vb1YN6XI
KKpmbbFARwGkhQ5O3jo3Kco1X5RxNdaQWndBgn4dfK1guhOQpsMCll2ClmfA/XgY
DsBpkqphV2aZ8w74IZ+3iUn5ES1yCPRJIghCyC0wR7rgJGalsF/XE1nLdFxvU50b
3sQH0pm7
-----END CERTIFICATE-----
`

	serverCertAppsCodeTest = `-----BEGIN CERTIFICATE-----
MIIDvDCCAaQCAQEwDQYJKoZIhvcNAQELBQAwZjELMAkGA1UEBhMCQkQxEzARBgNV
BAgMClNvbWUtU3RhdGUxDjAMBgNVBAcMBURoYWthMRUwEwYDVQQKDAxhcHBzY29k
ZS5jb20xGzAZBgNVBAMMEmh0dHAuYXBwc2NvZGUudGVzdDAeFw0xNzEwMTAwNjU0
MjRaFw0xODEwMTAwNjU0MjRaMGYxCzAJBgNVBAYTAkJEMRMwEQYDVQQIDApTb21l
LVN0YXRlMQ4wDAYDVQQHDAVEaGFrYTEVMBMGA1UECgwMYXBwc2NvZGUuY29tMRsw
GQYDVQQDDBJodHRwLmFwcHNjb2RlLnRlc3QwgZ8wDQYJKoZIhvcNAQEBBQADgY0A
MIGJAoGBALKGvoHy3hR44b9ZWKKt59acdEtjPXC5ijAcmFR1hNkyVu8seFxVHat3
4Bq4De76wo0Lb2JV5f6Wlic6/57eujQTXBqBJ+UYnyKD5L4x8xZWtpwbYDCiXqhu
fR2/8L6/xEsRWt4PEUxzhkjQjtj3EgAJ6rcNP8a1jGe4RC6mDF1TAgMBAAEwDQYJ
KoZIhvcNAQELBQADggIBAHg7m2BH52jx2VM0sVRCDKGCfuhngwxn5gEB5srlcAf4
16p12AQK7QFh/ghOzvwhcd0yPs+jU6YohcKfTi1vnwCIuOOqqBQmLIAJQth2aEh5
639hEYKrLiCljX4J+paj4cDwkYlqYnE6PoPEzP6yV9wCTd+lUZScBS1uGqNn1thk
/1T+gozcsVt37ru2rd9K9iBDSVbbv5t3QtaHqCfvuYk41X0p6p/HXW3M0FVRT9cr
bcxIPF88ZOM5y0GJPQ6QSWF8CaO1nb//4Cu1roVtdNpOOS8KtU/ItU5RIny67tlm
DBFG0LhJbZ4zU+zqOxwuT45LXgzC5wdrjIqmrNE09+gAJXuk2Ix+rxy+LU6fxbeu
kWBSGDqKDrSAkbsD3kb05CSJwzJd+lhkFzBHg/7tq0n6qIjx66NFjxuCjaKwNsIY
i1+vuvEaVDjSX8ku2e1gvGsK8HxfKihcuS2hswFuoAZZ5Fw9wj286A8Kfu14bXFC
tY66p1mu3RkzT7h7hZfkidzSGPviC9YGbO4TeldAO0L/6LWOBifrA4GBwVGmIr6r
jjlqQT1Xr86CRwJaBzTi9BSiwL047l2+136M2keMZ+H1Akt0hh3rVJkR7q8ZAT3m
LGmBdX1VEAqtO4dOMjI7D/CZigkmPndT0XMtIZlG1IFMwixDfNFfGfXyDuwigawe
-----END CERTIFICATE-----
`
	serverKeyAppsCodeTest = `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: DES-EDE3-CBC,8C60E8DB1AEE63C3

+wSmjwhG5sNuhyp8t8Dk0F96LC0fZEBVuoS2obI7NW9zjSqoM2k0MsEpex77QHAf
z0V9N6Sr0T0jPKJRh5zvSK85ZUMRYssBWPZX4IWuNMeCFJx9vzxIIjyhFj1tY6Cl
HAdQgQgOc5H+FCPGlB5ChipWnr5zv9ex0AE2vMpblQf1nwgJUrbx9sPuTDibwyBL
At0AhaTAlR4X/louew2jYfB3JlECr7Fl4kpqLuJcaBEqepNHjZVv8pSoLilAqE5I
VTeSXDkfU1PRMxCctAbzbQFQ9+7cEq4m0vFmlU8/76PxPZ3SM6z0fWjCAmony7VP
ZidIS/V9riCne55WBPSSNtgPao/24onO7/7HGWPnHr+emW4PtVSUOj5PPf688Jnv
OKTbwQRFv/Yrht27wmPLegk7juUAyGeeyUdkOvPU1DbztVkkhK9JPL12BBTvdW7n
MOzXwGe20J1kXfiKlHymMJWoN3o3D5lmhMvox1O1J0scmT668jF8oiIQXK9BS/Tx
Z+gA9At7hwCQt+c2zYlDEK3i/qAyT4gmcMgXGG4ZzD/6g+eg46DmwQ7c4sVP91NL
q669cfKdQIFGgqYSk1/sPg8GWhwDrowNCg5PbErC58to327lj0nwcMbX1vM2InRB
O4os2FZiCFno5D/8zQCoTP2GcRNcTOgiPDprgrgPkK3NHMpGyorB03b/y52mefb3
+CppoSaDDcKTwYYRG0ACjYz+Kp7DZx8EZyLRC5Qxdnt4AZkY5Cl8AanJeInEwFYi
Z2C8ffPlBlmXUaqron+3kpKPSprgv7MVADCIGIqDc1KLitIiK6wwRA==
-----END RSA PRIVATE KEY-----
`
	clientCertAppsCodeTest = `-----BEGIN CERTIFICATE-----
MIIDvDCCAaQCAQEwDQYJKoZIhvcNAQELBQAwZjELMAkGA1UEBhMCQkQxEzARBgNV
BAgMClNvbWUtU3RhdGUxDjAMBgNVBAcMBURoYWthMRUwEwYDVQQKDAxhcHBzY29k
ZS5jb20xGzAZBgNVBAMMEmh0dHAuYXBwc2NvZGUudGVzdDAeFw0xNzEwMTAwNjU1
NDVaFw0xODEwMTAwNjU1NDVaMGYxCzAJBgNVBAYTAkJEMRMwEQYDVQQIDApTb21l
LVN0YXRlMQ4wDAYDVQQHDAVEaGFrYTEVMBMGA1UECgwMYXBwc2NvZGUuY29tMRsw
GQYDVQQDDBJodHRwLmFwcHNjb2RlLnRlc3QwgZ8wDQYJKoZIhvcNAQEBBQADgY0A
MIGJAoGBAN0Pvm/j5G6kVlZF9rbGUuRO2K2ZTK+iPep4NDchmH1M7MJdKIdkJOky
uazqlEIo/GOGnjmU32heFYkndzzYyLt6h8v1QA2uKWTvwD4oUwaQFI74By7X7KfR
oNDG3Tm9jTHl1eKO1CDvicNEQ+xpLqrT+YwggzXZhTRX6NX6MefjAgMBAAEwDQYJ
KoZIhvcNAQELBQADggIBAED4rlUVZz1MZNuUIGWcdwql/7Sx1KXgolTiL6KVctPO
U/LftYEftiiJIwdGyT2dF/OyPK0+mrayljblGkA5wSKRxrKquo7yzEo/rEzAOZMk
zCVlncuyZma5o3wQ43RRTh9krTR88nZLLIGwkSwB6bJ/sBM+152fI+7bAbrpgDkl
yQlTFhBWdrMbTfa1zFv8sSUBGmlCNPuqcElLWod+uXzbUH4RIen5rb2msUxfjZKz
aLoLQ/ndhTLAhLrrFJRn9WCJwc6N8n4hwD4dmoS6GFAaJYyRWaV2zfKT9QIPehXX
DQY3K6s4dvhgiEY1gkXYwO2WJ8YM508Cjy6CJhJttpHR11WJiWkC/g9rnxehF2QM
gBL+a5mn/Z6xMa95uRm7zWz8DPZk2rzQ/huTCEL6RYKnjsJGjBVNWNR+NdvsEKpx
fegmQ5WbtWEXHYzwy9cm6jgxzquHo4ikTBDD/Z9Oxj2o/cwS64cKp0KeZqnPboWk
FoTLGlLOLub4mtuO1fGNUq9ZRJLJm81Nk3JuSGzu71YPzpR6bRBRXJGxMEN6/uEs
UFtnjJfs7Xkds7jH/XGCGnUGToZASqxYBz77NKoCXkOCkHYyVnCq7qurA98TcBye
XE6i7zrfvzYiq0eNM6sQhbG9+iLw9vmUUQeS8Ri9pnt/kpy/Q52a09H9G5Ch10Gl
-----END CERTIFICATE-----
`
	clientKeyAppsCodeTest = `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: DES-EDE3-CBC,58A8E237B030C11A

A8UhkRrmdOtUFOM5O8qqFrECXAYHy7KJKvZYV9ie0w3GJqDVf4iJCNZBKzLI/wiF
aQBczbfe1WudbvxVAYcHhmuqBMBjG6B+c7NTkg3zM7Q0KNMVECQNA47hI7viO3o+
dX5YFOWGKgmNSk6IMeDSiONw6BMq7IvMavMEk+eL/eVGIY2KMspWJBFiTDsu3F1m
XMVl2vccIjarNlZlkKEaZlBbKO/XvPG7Gpbk8T/rvTKDWwkKqLk7VG6J3C2YhxMv
Wtg7NQwr/92i8SvOzyALBk4A5unxmQZrKgXRX3pxD9PBdz8AWSmIfVH5vBiXkica
I9pD+u8ww1GUX0QBUZev746stm77dCi4lB0qSSKYWBMnDhzPd477VF3KYH/ouZuy
l8kjBa9IUEvd6oyhMA7K+tVZcexNaxYcA+nrWLu9wlMy54dtqJ9ghj7Puee+802H
Sm15XCHQC1RX8MeGYp7wn0QadPWebEKO2QLYzXPUZRHRfZh8HfFisCAUa/HIRZI2
+Ys7TnFQyFiyF0MpDvh7nsfn6+Zj1wUfq/VTAxlmMfxipFqxYMI9rLYecTOZeAit
xFx7x+5K8Oxgn8gQZVleR073EG04xUiyvNmtAehVcu/E8tGkdo+qAmxnB9pWNYPw
XvO3oy0ayj7g+VWn8049MmJdE7x6PsPRUzIMKPFE/jYmuC847uTv6d3NPAmV39eT
DmE8VZwKq3kq8sf9wxIuZ3cVPQ5bZbdPQaR0Hho7lADvdTkdz9Vs0dk0kM4W7cH8
zDKNTn541As31SJw3WstIwBKr7UAtAx7CZfh/qo8Fx6PSo/QdO5e8g==
-----END RSA PRIVATE KEY-----
`
)
