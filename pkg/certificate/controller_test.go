package certificate

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/appscode/log"
	api "github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	acf "github.com/appscode/voyager/client/clientset/fake"
	"github.com/appscode/voyager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/acme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

var (
	fakeConfig = &rest.Config{}
)

func TestLoadProviderCredential(t *testing.T) {
	cert := &api.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.CertificateSpec{
			ProviderCredentialSecretName: "foosecret",
		},
	}

	fakeController := NewController(fakeConfig, fake.NewSimpleClientset(), acf.NewFakeExtensionClient(), config.Options{SyncPeriod: time.Second * 5}, cert)
	fakeController.acmeClientConfig = &ACMEConfig{
		ProviderCredentials: make(map[string][]byte),
	}

	fakeController.loadProviderCredential()
	assert.Equal(t, len(fakeController.acmeClientConfig.ProviderCredentials), 0)

	fakeSecret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foosecret",
			Namespace: "bar",
		},
		Data: map[string][]byte{
			"foo-data": []byte("foo-data"),
		},
	}

	s, err := fakeController.KubeClient.CoreV1().Secrets("bar").Create(fakeSecret)
	assert.Nil(t, err)
	assert.Equal(t, "foosecret", s.Name)
	assert.Equal(t, "bar", s.Namespace)
	log.Debugln("Secret Created.", *s)

	fakeController.loadProviderCredential()
	assert.Equal(t, len(fakeController.acmeClientConfig.ProviderCredentials), 1)
	assert.Equal(t, string(fakeController.acmeClientConfig.ProviderCredentials["foo-data"]), "foo-data")
	log.Debugln("Provider credential", fakeController.acmeClientConfig.ProviderCredentials)
}

func TestEnsureClient(t *testing.T) {
	if testing.Verbose() {
		skipTestIfSecretNotProvided(t)
		cert := &api.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: api.CertificateSpec{
				Domains:                      strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				Email:                        os.Getenv("TEST_ACME_USER_EMAIL"),
				Provider:                     "googlecloud",
				ProviderCredentialSecretName: "fakesecret",
			},
		}
		fakeController := NewController(fakeConfig, fake.NewSimpleClientset(
			&apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bar"},
			},
		), acf.NewFakeExtensionClient(), config.Options{SyncPeriod: time.Second * 5}, cert)

		fakeController.acmeClientConfig = &ACMEConfig{
			Provider:            "googlecloud",
			ProviderCredentials: make(map[string][]byte),
			UserDataMap:         make(map[string][]byte),
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
		_, err := fakeController.KubeClient.CoreV1().Secrets("bar").Create(fakeSecret)
		assert.Nil(t, err)

		fakeController.ensureACMEClient()
		secret, err := fakeController.KubeClient.CoreV1().Secrets("bar").Get(defaultUserSecretPrefix+cert.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, 1, len(secret.Data))
	}
}

func TestFakeRegisterACMEUser(t *testing.T) {
	cert := &api.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: api.CertificateSpec{
			Domains:                      []string{"example.com"},
			Email:                        newFakeACMEUser().email,
			Provider:                     "googlecloud",
			ProviderCredentialSecretName: "fakesecret",
		},
	}
	fakeController := NewController(fakeConfig, fake.NewSimpleClientset(
		&apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bar"},
		},
	), acf.NewFakeExtensionClient(), config.Options{SyncPeriod: time.Second * 5}, cert)

	acmeClient := &ACMEClient{
		Client: newFakeACMEClient(),
	}
	if acmeClient.Client != nil {
		fakeController.acmeClientConfig = &ACMEConfig{
			UserData: &ACMEUserData{
				Email:        newFakeACMEUser().email,
				Registration: newFakeACMEUser().regres,
				Key:          x509.MarshalPKCS1PrivateKey(newFakeACMEUser().privatekey),
			},
		}
		err := fakeController.registerACMEUser(acmeClient)
		if !assert.NotNil(t, err) {
			assert.Nil(t, err)
			secret, err := fakeController.KubeClient.CoreV1().Secrets("bar").Get(defaultUserSecretPrefix+cert.Name, metav1.GetOptions{})
			assert.Nil(t, err)
			if assert.NotNil(t, secret) {
				assert.Equal(t, 1, len(secret.Data))
			}
		}
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
				Domains:                      strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				Email:                        os.Getenv("TEST_ACME_USER_EMAIL"),
				Provider:                     "googlecloud",
				ProviderCredentialSecretName: "fakesecret",
			},
		}
		fakeController := NewController(fakeConfig, fake.NewSimpleClientset(), acf.NewFakeExtensionClient(), config.Options{SyncPeriod: time.Second * 5}, cert)
		fakeController.ExtClient.Certificates("bar").Create(cert)

		fakeController.acmeClientConfig = &ACMEConfig{
			ProviderCredentials: make(map[string][]byte),
			UserDataMap:         make(map[string][]byte),
			Provider:            "googlecloud",
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
		_, err := fakeController.KubeClient.CoreV1().Secrets("bar").Create(fakeSecret)
		assert.Nil(t, err)

		fakeController.create()

		secret, err := fakeController.KubeClient.CoreV1().Secrets("bar").Get(defaultUserSecretPrefix+cert.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, len(secret.Data), 1)

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
		value, ok := secret.Annotations[certificateKey]
		assert.True(t, ok)
		assert.Equal(t, "true", value)

		certificate, err := fakeController.ExtClient.Certificates("bar").Get("foo")
		if err != nil {
			t.Fatal(err)
		}
		log.Infoln(certificate.Status)
		log.Infoln(certificate.Status.Details)
	}
}

func TestDemoCertificates(t *testing.T) {
	c := &api.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-do-token",
			Namespace: "default",
		},
		Spec: api.CertificateSpec{
			Domains:  []string{"john.example.com"},
			Provider: "digitalocean",
			Email:    "john@example.com",
			ProviderCredentialSecretName: "mysecret",
		},
	}

	w := bytes.NewBuffer(nil)
	err := acs.ExtendedCodec.Encode(c, w)
	assert.Nil(t, err)
	assert.NotEqual(t, 0, len(w.String()))
	assert.Equal(t, "Certificate", c.TypeMeta.Kind)
}

type mockUser struct {
	email      string
	regres     *acme.RegistrationResource
	privatekey *rsa.PrivateKey
}

func (u mockUser) GetEmail() string                            { return u.email }
func (u mockUser) GetRegistration() *acme.RegistrationResource { return u.regres }
func (u mockUser) GetPrivateKey() crypto.PrivateKey            { return u.privatekey }

type directory struct {
	NewAuthzURL   string `json:"new-authz"`
	NewCertURL    string `json:"new-cert"`
	NewRegURL     string `json:"new-reg"`
	RevokeCertURL string `json:"revoke-cert"`
}

type challenge struct {
	Type   acme.Challenge `json:"type,omitempty"`
	Status string         `json:"status,omitempty"`
	URI    string         `json:"uri,omitempty"`
	Token  string         `json:"token,omitempty"`
}

func newFakeACMEUser() mockUser {
	keyBits := 32 // small value keeps test fast
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		log.Fatal("Could not generate test key:", err)
	}
	user := mockUser{
		email:      "test@test.com",
		regres:     new(acme.RegistrationResource),
		privatekey: key,
	}
	return user
}

func newFakeACMEClient() *acme.Client {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal stub ACME server for validation.
		w.Header().Add("Replay-Nonce", "12345")
		w.Header().Add("Retry-After", "0")
		switch r.Method {
		case "HEAD":
		case "POST":
			writeJSONResponse(w, &challenge{Type: "http-01", Status: "valid", URI: "http://example.com/", Token: "token"})

		case "GET":
			data, _ := json.Marshal(directory{NewAuthzURL: "http://test", NewCertURL: "http://test", NewRegURL: "http://test", RevokeCertURL: "http://test"})
			w.Write(data)
		default:
			http.Error(w, r.Method, http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()
	keyType := acme.RSA2048
	client, err := acme.NewClient(ts.URL, newFakeACMEUser(), keyType)
	if err != nil {
		log.Fatalf("Could not create client: %v", err)
	}
	return client
}

// writeJSONResponse marshals the body as JSON and writes it to the response.
func writeJSONResponse(w http.ResponseWriter, body interface{}) {
	bs, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(bs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
