package certificates

import (
	"bytes"
	"fmt"
	"testing"

	flags "github.com/appscode/go-flags"
	aci "github.com/appscode/k8s-addons/api"
	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/k8s-addons/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
)

func init() {
	flags.SetLogLevel(10)
}

func TestLoadProviderCredential(t *testing.T) {
	fakeController := NewController(fake.NewFakeClient(), fake.NewFakeExtensionClient())
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

	_, err := fakeController.KubeClient.Core().Secrets("bar").Create(fakeSecret)
	assert.Nil(t, err)

	fakeController.loadProviderCredential()
	assert.Equal(t, len(fakeController.acmeClientConfig.ProviderCredentials), 1)
	assert.Equal(t, string(fakeController.acmeClientConfig.ProviderCredentials["foo-data"]), "foo-data")
}

func TestEnsureClient(t *testing.T) {
	fakeController := NewController(fake.NewFakeClient(), fake.NewFakeExtensionClient())
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

	fakeController.ensureACMEClient()

	secret, err := fakeController.KubeClient.Core().Secrets("bar").Get(fakeController.certificate.Name)
	assert.Nil(t, err)
	assert.Equal(t, len(secret.Data), 1)

	fmt.Println(string(secret.Data["user-info"]))
}

func TestCreate(t *testing.T) {
	fakeController := NewController(fake.NewFakeClient(), fake.NewFakeExtensionClient())
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

func TestDemoCertificates(t *testing.T) {
	c := &aci.Certificate{
		ObjectMeta: api.ObjectMeta{
			Name:      "test-do-token",
			Namespace: "default",
		},
		Spec: aci.CertificateSpec{
			Domains:  []string{"sadlil.containercloud.io"},
			Provider: "digitalocean",
			Email:    "sadlil@appscode.com",
			ProviderCredentialSecretName: "mysecret",
		},
	}
	w := bytes.NewBuffer(nil)
	err := acs.ExtendedCodec.Encode(c, w)
	if err == nil {
		fmt.Println(w.String())
	}
}
