package e2e

import (
	"testing"
	"time"

	"github.com/appscode/go/runtime"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/haproxy"
	"github.com/appscode/voyager/pkg/operator"
	"github.com/appscode/voyager/test/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

const (
	TestTimeout = 2 * time.Hour
)

var (
	root       *framework.Framework
	invocation *framework.Invocation
)

func RunE2ETestSuit(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TestTimeout)

	root = framework.New()
	invocation = root.Invoke()

	junitReporter := reporters.NewJUnitReporter("report.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Voyager E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	op := operator.New(
		root.KubeClient,
		root.CRDClient,
		root.V1beta1Client,
		nil,
		config.Options{
			CloudProvider: root.Config.CloudProviderName,
			HAProxyImage:  root.Config.HAProxyImageName,
			IngressClass:  root.Config.IngressClass,
		},
	)

	By("Ensuring Test Namespace " + root.Config.TestNamespace)
	err := root.EnsureNamespace()
	Expect(err).NotTo(HaveOccurred())

	if !root.Config.InCluster {
		By("Running Controller in Local mode")
		err := op.Setup()
		Expect(err).NotTo(HaveOccurred())

		err = haproxy.LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
		Expect(err).NotTo(HaveOccurred())

		go op.Run()
	}
	root.EventuallyCRD().Should(Succeed())

	Eventually(invocation.Ingress.Setup).Should(BeNil())
})

var _ = AfterSuite(func() {
	if root.Config.Cleanup {
		root.DeleteNamespace()
		invocation.Ingress.Teardown()
	}
})

const (
	// Following is a fake SSL certificate data, generated for test purposes only.
	fakeHTTPAppsCodeTestCert = `-----BEGIN CERTIFICATE-----
MIIDDTCCAfWgAwIBAgIJAMxyA+toDT7HMA0GCSqGSIb3DQEBBQUAMB0xGzAZBgNV
BAMMEmh0dHAuYXBwc2NvZGUudGVzdDAeFw0xNzEwMTAwNDQwMTZaFw0yNzEwMDgw
NDQwMTZaMB0xGzAZBgNVBAMMEmh0dHAuYXBwc2NvZGUudGVzdDCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBALyPkeXJOXwrmRHo4ApxaN0rodtdXlpib1Vk
BaZ9FOiXk2/Kzgz71Ab0JQLObPf/Lcfx2SLH2TrMb2grHZVIu45Ppd9/SDg794LJ
lhJgSjTqSpz4x0poHDd0ru2Jkk3PmS3kNqPvPOXSsJUYS5VkX8/TaCNjGubDxZzT
XTgDx6X97a/ypo4xU778zxOnut0RmeqGDb8dkb1lofumKGvbbIp5Sf17+w5i5ri5
J+cJh/+vR8F/kCrLB7eaAQSs4iESMlwmh6cLoRozgDJHR/laSFPLoS1QnlzBMBS0
VyKlGWK0TQXD/qm/qw+jDBQ/jLsQb5osacpy/YOB0WGef0WsHcECAwEAAaNQME4w
HQYDVR0OBBYEFCdkSccKeT45KWFtVMHdCS9dYZUTMB8GA1UdIwQYMBaAFCdkSccK
eT45KWFtVMHdCS9dYZUTMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEB
AE9ajwsyT+69yFByFcgrEcORNcjZUlZQGqz9jNC8w+/NJkbcRHCLTWOKGFXPyBHd
DRQxkSsmCMb71TefQZNTp1Q5IbOhhyOru9ejHa/893xQ9ktIEt+ZGQoylBd8lO7s
NxRNrrE5+tpHK9Aa3vZH8YtBsRjVVNQDkbi7HyWg0LB2GlMuvCvW+ISnAt54Pghz
CKOAL7T61qFM7khRPrmfkcO0DIjzkr8ckMoKEbGvVlgwAC1J4Z06Ifo88ke8sAbH
ObF+sLLZANdiSFI0mR38KeVvIKg87zDiI17Cwfj6mzOfV2dlFTBv4Nsc8KKwNJFJ
HasLCzr1hTDEu24wc42TaIw=
-----END CERTIFICATE-----
`
	fakeHTTPAppsCodeTestKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAvI+R5ck5fCuZEejgCnFo3Suh211eWmJvVWQFpn0U6JeTb8rO
DPvUBvQlAs5s9/8tx/HZIsfZOsxvaCsdlUi7jk+l339IODv3gsmWEmBKNOpKnPjH
SmgcN3Su7YmSTc+ZLeQ2o+885dKwlRhLlWRfz9NoI2Ma5sPFnNNdOAPHpf3tr/Km
jjFTvvzPE6e63RGZ6oYNvx2RvWWh+6Yoa9tsinlJ/Xv7DmLmuLkn5wmH/69HwX+Q
KssHt5oBBKziIRIyXCaHpwuhGjOAMkdH+VpIU8uhLVCeXMEwFLRXIqUZYrRNBcP+
qb+rD6MMFD+MuxBvmixpynL9g4HRYZ5/RawdwQIDAQABAoIBAAb0QIw59J3Iudd4
QDMCZbyqbEi708v/j12V18OBH3FIjc50q069Rt+Ox4Kn/ErVJWoXWEu5FSDfA0jT
Nj8YNJqWA2cPuakhRQqUxq0c0f/LmD7byfXLiybcbcsi5Ltr6ZlQrlczboqHT63f
/IGg9wuiH1gWwpo6JCKZhPmY4hcUCw8uu3jmiA8nCDioCiD9EVaIlI3H/4eRGb1q
/LKJOR7EY378NeDIqtR/+Lh0MU1g55Ue9Bdjv+NmoODXxaYmhhGlMn+wwSsBuHLj
nHyPge0AB5K5FHhE6ew3GLSIihUBqiQ/vhm/RN9uuoCgoLo1xwREZS7OtyBINSam
1w5AvkUCgYEA9UfsJZZxNl7DRlT9JFPEDBkziBW9QHMSkmUAsc00mZUzPJO1tAVM
VnUT4JyDNICK6wOb4KoGb7Luk9BHH/NCuc/+9B1Oje4SKVAZ+WsiEvYLgPNj1DlN
NG08OjEvVuiP8qcVpyzo7Y8PUBu26Cj3zxmwknRDoQe1YTqfc4578m8CgYEAxM0X
fKoB5KNdt83/IdA/AkMV0p/kJovR6pGXXUATovqfGajuiksELMp/jIPwzElrpe1j
nDCbJqv2feO56PPbGTRtd0UvDlNUxDsI1PH+U5rDWTxKt5DuFZcM1AGIjDCpr6RH
xxJaimBQlxL/wuFEqgeW6HlBDmxnxYkuhv4RSs8CgYB3mnOfPIXGAl1sLUMm9KWz
VJKZOCiJhdM3iYLWMH8GqQdL8ab3umGoAv0HWKpt7oRO5vqaia4Lx4+oijY0cTVH
UBI9TREiCkXW2VVhFwmNf2bKoWQ7dxmbh+yHX7Z6xXpz01+uniqStGC+KlV9TYTQ
+vDr6T+VBSI/4AsimQb9hwKBgEzi434s3Th6Kq8Yp8iKF1PG6cuz8+qrTYObBcvE
sOdHisj3mtoknKjzJAm9smHdfVUB/ZyT0Mm2/UIJqiQ8wSiDtxCV0uCB5egUOEsZ
kAcRu6gtSfOVh66fqL9bKgG7MVARmolHvl+5aULchVeZsr3K4UZJuQTtjU07XxYW
RKM5AoGAUvokupo7yoeTskKLzU0s4bzR02HnK/Fo8meR8pC9dmCzO3ydr+4XVVyL
plDItRyLcCFBpkDl4mkSt4Qh3l/r00GMpxXtSK7vMqDVkF3HJmd4Xb3aXjZAvYwd
NH6F6E4EgTic+00cjxURy9aFwLvPGAW+BrjTa6hqG/bn2fAAvAo=
-----END RSA PRIVATE KEY-----`
)
