package certificates

import (
	"bytes"
	"os"
	"strings"
	"testing"

	aci "github.com/appscode/k8s-addons/api"
	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/k8s-addons/client/clientset/fake"
	"github.com/appscode/log"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestLoadProviderCredential(t *testing.T) {
	fakeController := NewController(fakeclientset.NewSimpleClientset(), fake.NewFakeExtensionClient())
	fakeController.certificate = &aci.Certificate{
		ObjectMeta: api.ObjectMeta{
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

	fakeSecret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
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
			&api.Secret{
				ObjectMeta: api.ObjectMeta{Name: "secret", Namespace: "bar"},
			},
		), fake.NewFakeExtensionClient())
		fakeController.certificate = &aci.Certificate{
			ObjectMeta: api.ObjectMeta{
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

		fakeSecret := &api.Secret{
			ObjectMeta: api.ObjectMeta{
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

/*func TestCreate(t *testing.T) {
	fakeController := NewController(fakeclientset.NewSimpleClientset(), fake.NewFakeExtensionClient())
	fakeController.certificate = &aci.Certificate{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: aci.CertificateSpec{
			Domains:                      []string{"sadlil.appscode.co"},
			Email:                        "sadlil@appscode.com",
			Provider:                     "googlecloud",
			ProviderCredentialSecretName: "fakesecret",
		},
	}
	fakeController.ACExtensionClient.Certificate("bar").Create(fakeController.certificate)

	fakeController.acmeClientConfig = &ACMEConfig{
		ProviderCredentials: make(map[string][]byte),
		UserDataMap:         make(map[string][]byte),
	}

	fakeController.acmeClientConfig = &ACMEConfig{
		Provider: "googlecloud",
	}

	fakeSecret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name:      "fakesecret",
			Namespace: "bar",
		},
		Data: map[string][]byte{
			"GCE_PROJECT": []byte("tigerworks-kube"),
		},
	}
	_, err := fakeController.KubeClient.Core().Secrets("bar").Create(fakeSecret)
	assert.Nil(t, err)

	fakeController.create()

	secret, err := fakeController.KubeClient.Core().Secrets("bar").Get(fakeController.certificate.Name)
	assert.Nil(t, err)
	assert.Equal(t, len(secret.Data), 1)

	list, err := fakeController.KubeClient.Core().Secrets("").List(api.ListOptions{})
	if err == nil {
		for _, item := range list.Items {
			fmt.Println(item.Name, item.Namespace)
		}
	}

	// Check the certificate data
	secret, err = fakeController.KubeClient.Core().Secrets("bar").Get("cert-" + fakeController.certificate.Name)
	assert.Nil(t, err)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(secret.Data), 3)
	fmt.Println(string(secret.Data[fakeController.certificate.Name+"."+fakeController.certificate.Namespace+".crt"]))

	certificate, err := fakeController.ACExtensionClient.Certificate("bar").Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(certificate.Status)
	fmt.Println(certificate.Status.Details)
}
*/

func TestDemoCertificates(t *testing.T) {
	c := &aci.Certificate{
		ObjectMeta: api.ObjectMeta{
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
