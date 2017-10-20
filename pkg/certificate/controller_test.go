package certificate

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acf "github.com/appscode/voyager/client/fake"
	"github.com/appscode/voyager/pkg/config"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureClient(t *testing.T) {
	if testing.Verbose() {
		skipTestIfSecretNotProvided(t)

		fakeSecret := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fakesecret",
				Namespace: "bar",
			},
			Data: map[string][]byte{
				"GCE_PROJECT":              []byte(os.Getenv("TEST_GCE_PROJECT")),
				"GCE_SERVICE_ACCOUNT_DATA": []byte(os.Getenv("TEST_GCE_SERVICE_ACCOUNT_DATA")),
			},
		}

		cert := &api.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: api.CertificateSpec{
				Domains:            strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				ACMEUserSecretName: "user",
				ChallengeProvider:  api.ChallengeProvider{DNS: &api.DNSChallengeProvider{Provider: "googlecloud", CredentialSecretName: "fakesecret"}},
				Storage:            api.CertificateStorage{Secret: &apiv1.LocalObjectReference{}},
			},
		}

		fakeController, err := NewController(etx.Background(), fake.NewSimpleClientset(
			&apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bar"},
			},
			&apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "bar"},
				Data: map[string][]byte{
					api.ACMEUserEmail: []byte(os.Getenv("TEST_ACME_USER_EMAIL")),
					api.ACMEServerURL: []byte(LetsEncryptStagingURL),
				},
			},
			fakeSecret,
		), acf.NewSimpleClientset().VoyagerV1beta1(), config.Options{ResyncPeriod: time.Second * 5}, cert)
		assert.Nil(t, err)

		err = fakeController.getACMEClient()
		assert.Nil(t, err)

		secret, err := fakeController.KubeClient.CoreV1().Secrets("bar").Get("user", metav1.GetOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, 2, len(secret.Data))
	}
}

func TestCreate(t *testing.T) {
	if testing.Verbose() {
		skipTestIfSecretNotProvided(t)
		cert := &api.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: api.CertificateSpec{
				Domains:            strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				ACMEUserSecretName: "user",
				ChallengeProvider:  api.ChallengeProvider{DNS: &api.DNSChallengeProvider{Provider: "googlecloud", CredentialSecretName: "fakesecret"}},
				Storage:            api.CertificateStorage{Secret: &apiv1.LocalObjectReference{}},
			},
		}

		fakeUser := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "bar"},
			Data: map[string][]byte{
				api.ACMEUserEmail: []byte(os.Getenv("TEST_ACME_USER_EMAIL")),
				api.ACMEServerURL: []byte(LetsEncryptStagingURL),
			},
		}

		fakeSecret := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fakesecret",
				Namespace: "bar",
			},
			Data: map[string][]byte{
				"GCE_PROJECT":              []byte(os.Getenv("TEST_GCE_PROJECT")),
				"GCE_SERVICE_ACCOUNT_DATA": []byte(os.Getenv("TEST_GCE_SERVICE_ACCOUNT_DATA")),
			},
		}

		fakeController, err := NewController(etx.Background(), fake.NewSimpleClientset(fakeUser, fakeSecret), acf.NewSimpleClientset().VoyagerV1beta1(), config.Options{ResyncPeriod: time.Second * 5}, cert)
		if assert.Nil(t, err) {
			err = fakeController.Process()
			assert.Nil(t, err)

			secret, err := fakeController.KubeClient.CoreV1().Secrets("bar").Get("user", metav1.GetOptions{})
			assert.Nil(t, err)
			assert.Equal(t, len(secret.Data), 2)

			list, err := fakeController.KubeClient.CoreV1().Secrets("").List(metav1.ListOptions{})
			if err == nil {
				for _, item := range list.Items {
					log.Infoln("List for Secrets that created", item.Name, item.Namespace)
				}
			}

			// Check the certificate data
			secret, err = fakeController.KubeClient.CoreV1().Secrets("bar").Get("cert-"+cert.Name, metav1.GetOptions{})
			assert.Nil(t, err)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, len(secret.Data), 2)

			cert, err = fakeController.VoyagerClient.Certificates("bar").Get("foo", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			log.Infoln(cert.Status)
			log.Infoln(cert.Status.LastIssuedCertificate)
		}
	}
}

func skipTestIfSecretNotProvided(t *testing.T) {
	if len(os.Getenv("TEST_GCE_PROJECT")) == 0 ||
		len(os.Getenv("TEST_GCE_SERVICE_ACCOUNT_DATA")) == 0 ||
		len(os.Getenv("TEST_ACME_USER_EMAIL")) == 0 ||
		len(os.Getenv("TEST_DNS_DOMAINS")) == 0 {
		t.Skip("Skipping Test, Secret Not Provided")
	}
}

func TestCertificateRenewTime(t *testing.T) {
	demoNotAfter := time.Now().Add(time.Hour * 24 * 6)
	res := demoNotAfter.After(time.Now().Add(time.Hour * 24 * 7))
	assert.Equal(t, res, false)

	demoNotAfter = time.Now().Add(time.Hour * 24 * 25)
	res = demoNotAfter.After(time.Now().Add(time.Hour * 24 * 7))
	assert.Equal(t, res, true)
}

const (
	testCertMultiDomain = `-----BEGIN CERTIFICATE-----
MIIDPDCCAiSgAwIBAgIJAIpp+gWuABI6MA0GCSqGSIb3DQEBCwUAMD8xCzAJBgNV
BAYTAlNMMRAwDgYDVQQIDAdXZXN0ZXJuMRAwDgYDVQQHDAdDb2xvbWJvMQwwCgYD
VQQLDANBQkMwHhcNMTcwODIyMDcxNDA0WhcNMjcwODIwMDcxNDA0WjA/MQswCQYD
VQQGEwJTTDEQMA4GA1UECAwHV2VzdGVybjEQMA4GA1UEBwwHQ29sb21ibzEMMAoG
A1UECwwDQUJDMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAllDFusm4
Bre/20b6XBTicFvp8J/TIPTSvJ5SpUcPrfoyPVQTEcVsezPnmYOa5sunsyuhQqnN
LUecYfgrsGvtUrVmUKGQXm5D8ybpPN0YA+oSuB3d21XW02+ZHsUI9wC/y+nVl4d8
HVYNA0D/3AEkSJzKZBgtHIY0szcDKa0o5byaO0QXG5EOIChfJtTg7XkOG5aHzELD
gRfUJVuq70aLMyKxXpPssmzvJtOe878uSQimBm1vYGr7ll3fhI4XEWgOU+uKL2Sz
GZpfIL41Wd0gh0FbKDe3X3tZ5CFsn3gHI9AyOThL5qA+5EHdTSBkMcyRrcw2zFOm
Xo/MpMiU+maIIQIDAQABozswOTAJBgNVHRMEAjAAMAsGA1UdDwQEAwIF4DAfBgNV
HREEGDAWggl0ZXN0MS5jb22CCXRlc3QyLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEA
hTqbF6T4E4jf1fmmO2D6GUIkPBRr54Bx5Sp3+a4igDzgpFCg8doC9GJD588Z7vt8
ZsiYyZpQcCYWRa/+i/voBqWLl0h1z9xlLU7FkPOnJG7Ww29rDq+DdsptxW7xGyLP
rNwOItn+lVnroFIsJeV9+r+ZWpUtvYPpkeyjBadGknqnk6hJ2ODozBrY9U2Uf65O
84brwiknmZxbxPhmTLH5PiYlJLOmbHRNIPHIzdISlSYeRJVF7dkaRnSxeEjux+DJ
83274kS4U+MjHUfyVqE9IK4qVCkV/pTpgyvn1gcyp2BF2h62xVwxdDHO//C0EZYw
HYKHTpHd5CCQE4WXPEB8aQ==
-----END CERTIFICATE-----
`
)

func TestDecodeCert(t *testing.T) {
	pemBlock, _ := pem.Decode([]byte(testCertMultiDomain))
	crt, err := x509.ParseCertificate(pemBlock.Bytes)
	if assert.Nil(t, err) {
		fmt.Println(crt.Subject.CommonName)
		fmt.Println(crt.DNSNames)
	}
}
