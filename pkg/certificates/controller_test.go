package certificates

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/client/clientset/fake"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/acme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestLoadProviderCredential(t *testing.T) {
	fakeController := NewController(fakeclientset.NewSimpleClientset(), fake.NewFakeExtensionClient())
	fakeController.certificate = &aci.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: aci.CertificateSpec{
			ProviderCredentialSecretName: "foosecret",
		},
	}
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

	s, err := fakeController.KubeClient.Core().Secrets("bar").Create(fakeSecret)
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
	if testframework.TestContext.Verbose {
		fakeController := NewController(fakeclientset.NewSimpleClientset(
			&apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bar"},
			},
		), fake.NewFakeExtensionClient())
		fakeController.certificate = &aci.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: aci.CertificateSpec{
				Domains:                      strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				Email:                        os.Getenv("TEST_ACME_USER_EMAIL"),
				Provider:                     "googlecloud",
				ProviderCredentialSecretName: "fakesecret",
			},
		}

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
		_, err := fakeController.KubeClient.Core().Secrets("bar").Create(fakeSecret)
		assert.Nil(t, err)

		fakeController.ensureACMEClient()
		secret, err := fakeController.KubeClient.Core().Secrets("bar").Get(defaultUserSecretPrefix + fakeController.certificate.Name)
		assert.Nil(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, 1, len(secret.Data))
	}
}

func TestFakeRegisterACMEUser(t *testing.T) {
	fakeController := NewController(fakeclientset.NewSimpleClientset(
		&apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bar"},
		},
	), fake.NewFakeExtensionClient())
	fakeController.certificate = &aci.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: aci.CertificateSpec{
			Domains:                      []string{"example.com"},
			Email:                        newFakeACMEUser().email,
			Provider:                     "googlecloud",
			ProviderCredentialSecretName: "fakesecret",
		},
	}

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
			secret, err := fakeController.KubeClient.Core().Secrets("bar").Get(defaultUserSecretPrefix + fakeController.certificate.Name)
			assert.Nil(t, err)
			if assert.NotNil(t, secret) {
				assert.Equal(t, 1, len(secret.Data))
			}
		}
	}
}

func TestCreate(t *testing.T) {
	if testframework.TestContext.Verbose {
		fakeController := NewController(fakeclientset.NewSimpleClientset(), fake.NewFakeExtensionClient())
		fakeController.certificate = &aci.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: aci.CertificateSpec{
				Domains:                      strings.Split(os.Getenv("TEST_DNS_DOMAINS"), ","),
				Email:                        os.Getenv("TEST_ACME_USER_EMAIL"),
				Provider:                     "googlecloud",
				ProviderCredentialSecretName: "fakesecret",
			},
		}
		fakeController.ExtClient.Certificate("bar").Create(fakeController.certificate)

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
		_, err := fakeController.KubeClient.Core().Secrets("bar").Create(fakeSecret)
		assert.Nil(t, err)

		fakeController.create()

		secret, err := fakeController.KubeClient.Core().Secrets("bar").Get(defaultUserSecretPrefix + fakeController.certificate.Name)
		assert.Nil(t, err)
		assert.Equal(t, len(secret.Data), 1)

		list, err := fakeController.KubeClient.Core().Secrets("").List(apiv1.ListOptions{})
		if err == nil {
			for _, item := range list.Items {
				log.Infoln("List for Secrets that created", item.Name, item.Namespace)
			}
		}

		// Check the certificate data
		secret, err = fakeController.KubeClient.Core().Secrets("bar").Get("cert-" + fakeController.certificate.Name)
		assert.Nil(t, err)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(secret.Data), 2)
		value, ok := secret.Annotations[certificateKey]
		assert.True(t, ok)
		assert.Equal(t, "true", value)

		certificate, err := fakeController.ExtClient.Certificate("bar").Get("foo")
		if err != nil {
			t.Fatal(err)
		}
		log.Infoln(certificate.Status)
		log.Infoln(certificate.Status.Details)
	}
}

func TestDemoCertificates(t *testing.T) {
	c := &aci.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-do-token",
			Namespace: "default",
		},
		Spec: aci.CertificateSpec{
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
