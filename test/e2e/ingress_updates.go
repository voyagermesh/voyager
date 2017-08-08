package e2e

import (
	"time"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
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
		Expect(f.Ingress.IsTargetCreated(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
		}
	})

	var (
		shouldUpdateServiceWhenSecretChanged = func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(80)))

			secret := &apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				Type: apiv1.SecretTypeTLS,
				StringData: map[string]string{
					"tls.key": fakeHTTPAppsCodeDevKey,
					"tls.crt": fakeHTTPAppsCodeDevCert,
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
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(443)))
		}
	)

	Describe("Secret Changed", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: "http.appscode.dev",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.IngressBackend{
										ServiceName: f.Ingress.TestServerName(),
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should update when secret changed", shouldUpdateServiceWhenSecretChanged)
	})
})

const (
	// Following is a fake SSL certificate data, generated for test purposes only.
	fakeHTTPAppsCodeDevCert = `-----BEGIN CERTIFICATE-----
MIIDCzCCAfOgAwIBAgIJAOaXTnfalwyQMA0GCSqGSIb3DQEBBQUAMBwxGjAYBgNV
BAMMEWh0dHAuYXBwc2NvZGUuZGV2MB4XDTE3MDcxODA5MTA0MFoXDTI3MDcxNjA5
MTA0MFowHDEaMBgGA1UEAwwRaHR0cC5hcHBzY29kZS5kZXYwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQCXn+4cxYbkFJ8qHrqORMPJ8a6/OtJooAwlsPWU
79z0kZ6RjBpw+hRRQvAxG4WPIpzqlhJcKAkQMOd5YlRZdoWi5P/fX+L5l8d2t1Yj
FnON/gZRvAX7alSvUBRdBFdZ/OJ6lDvVTWC+wYUnieePEmOnkd+ZopIaArLUEc3I
GljJRUG62srouOmTfbeCKdW5sI5R2UOo1pdrcxPN/J2lY6ixt8kneK80bosfpozu
9iVljWa7sO1s0Xsc/SwikDAIju8txpHEDl5SHcDX3JpVuNt9eeCquSuDNuegvjcH
RWzu/wHkcE7WGad7VkyXnzq1jBwBjryWINk3nzpmP7Q1BfnLAgMBAAGjUDBOMB0G
A1UdDgQWBBT08RnU4J5LD145GKdyMeRoWemOMjAfBgNVHSMEGDAWgBT08RnU4J5L
D145GKdyMeRoWemOMjAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBBQUAA4IBAQAv
pZFipxB65fuCZ4fV29Skxl4RwLWsvKRcKL7Fs+EyGhEF84B93N2jvwSO/fiifuHj
Q9algmNyftvEK5toHNIuGVSW35GpTGQ1GzNWlItlM5mmmXOK6kDvS8Yx4hszl8bz
ErhiVFmYp+huT7hI389VF5AIJ4Iqj6v0f1LKGa7jD2dJacFYWaHVV/z4W4LLvmKS
dxVm+Uu0HmX8D0vl+v2MHP/s7T20sx+VNcaw63HXeFmyn+EIa152jL1f12h2pB4t
4DZx5x7bvvGhTu/RktFl0rvT9vFkEOlmoy+ky4NlUDwyfLsRtXplQ2ltoyKvLge4
CstLLbiwGhfuzOGrsSD6
-----END CERTIFICATE-----
`
	fakeHTTPAppsCodeDevKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAl5/uHMWG5BSfKh66jkTDyfGuvzrSaKAMJbD1lO/c9JGekYwa
cPoUUULwMRuFjyKc6pYSXCgJEDDneWJUWXaFouT/31/i+ZfHdrdWIxZzjf4GUbwF
+2pUr1AUXQRXWfziepQ71U1gvsGFJ4nnjxJjp5HfmaKSGgKy1BHNyBpYyUVButrK
6Ljpk323ginVubCOUdlDqNaXa3MTzfydpWOosbfJJ3ivNG6LH6aM7vYlZY1mu7Dt
bNF7HP0sIpAwCI7vLcaRxA5eUh3A19yaVbjbfXngqrkrgzbnoL43B0Vs7v8B5HBO
1hmne1ZMl586tYwcAY68liDZN586Zj+0NQX5ywIDAQABAoIBABB/g244A/xvTf5M
R6pRSyh/Fq+SG/DscUXsolwpWVZ3PdTCdOIUI//Pk8kUII05i+9ukuLaLFpJp/Yq
P9lYLyRRXJIWoeDcpgSB4GqC9+HcYR2lotT/deV5hi202jhdbts9o+EKwVsgPXfW
5o5HxvYlxjm2WcVgw8qVgVmjnEOSDbvDgjb7yuCk9J2zIkYq+Qia+AzHqnIT4JmM
ZR+uxyQvhgwJQxXKMi/OqXL8AT4As8xBQb7L1FMXhmomyO1KAIz58DYUf+VCIXk1
S0Ama6sDg2yuJdDAk0mwFiJTlWbs4rzsBK9A9nGFZqCBGpCf7yjcuYhEk3giOoT8
qkszUIkCgYEAxcu89j2e+7/EUmAx6COtnWjFy3vx+JBMc8Hv1guze+qe4fTS7cvA
k3EHNjie+xXO2ZVfpGxpWFpUH/EH3Lo7dJfdPBRgqF9wSdpVlKVnQys6zw/aG5Ep
fEM2/NCBHDnWWqzW78/7I4/GSx0pVG5W8PkObv5vcCPUa9sclW+09nUCgYEAxD4N
EP93Drs19REIaCwZTJz4BMRmSCHA+Bfu0LdPEqTloVEv21zJZUiQt+e41wYwJRQK
7AUNl7leJJS3R34KCLZ9oRMhfOBU+2A5SHtg7j/Sx6UVCZhKFpjSJ/992qbJ4+4o
RASEMZ71WFKoVgHnT0Nhc4C2oBX+MQtT+C77pz8CgYBUdHTfs1oB5lTeU4Kbuzgz
YPwrsWWVG4/5UVKl02M0wu5KTq4NqRU2H2nT5gND9IDY+OXYoA2vEwqehN01izM9
ymZFc/H9kpqwfhBSovlffcLjjMI1SRssmsqM0j5+ndd/6hLwXJ7ABXDGu9Hc4iwv
Qji+fdd5S2M1Fl6zE/pxzQKBgQC0DH5uhwTUFj3GMC93bGZ13VrM/Oke6yEiPssU
4eqBn5szq8ptyC7bZ32nzcnQNtQ7YK04qNY0y5UtmOijhmdsYQrYmzXRXf16eWl1
MAXZ8eLQ24x2tivbmbDPk+EDmJ2JK3v0E/S5li9iLsxVxP9VwOuLTp/ANw12L/+F
qI2pfwKBgQCIJL+ltvMR1C75w2cW3v4xkC4fiV+kJ7GA0JMTftk9hws6iA620iWn
ciT4Bql5vJwULP7Sv+xLYK0tqnBE2dOzW23eAI5ZIlYiKDM9GGrRQvKIQmdRXSf1
oZmB+LUUEBO0+1+4QHcpbVlJlDLsv8cqcnLFpio4q+pFiAtuwq/G6w==
-----END RSA PRIVATE KEY-----`
)
