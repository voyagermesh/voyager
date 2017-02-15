package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/acme"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"github.com/appscode/go/flags"
)

func init() {
	flags.SetLogLevel(5)
	flags.InitFlags()
}

func TestNewDomainCollection(t *testing.T) {
	d := NewDomainCollection("a.com")
	assert.Equal(t, `["a.com"]`, d.String())

	d.Append("hello.world").Append("foo.bar")
	assert.Equal(t, `["a.com","hello.world","foo.bar"]`, d.String())
}

func TestACMECertData(t *testing.T) {
	certificateSecret := &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      defaultCertPrefix + "hello",
			Namespace: "default",
			Labels: map[string]string{
				certificateKey: "true",
				certificateKey + "/domains": NewDomainCollection("appscode.com").String(),
			},
			Annotations: map[string]string{
				certificateKey: "true",
			},
		},
		Data: map[string][]byte{
			api.TLSCertKey:       []byte("Certificate key"),
			api.TLSPrivateKeyKey: []byte("Certificate private key"),
		},
		Type: api.SecretTypeTLS,
	}

	cert, err := NewACMECertDataFromSecret(certificateSecret)
	assert.Nil(t, err)

	convertedCert := cert.ToSecret("hello", "default")
	assert.Equal(t, certificateSecret, convertedCert)
}

func TestACMECertDataError(t *testing.T) {
	certificateSecret := &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      defaultCertPrefix + "hello",
			Namespace: "default",
			Labels: map[string]string{
				certificateKey: "true",
				certificateKey + "/domains": NewDomainCollection("appscode.com").String(),
			},
			Annotations: map[string]string{
				certificateKey: "true",
			},
		},
		Data: map[string][]byte{
			api.TLSPrivateKeyKey: []byte("Certificate private key"),
		},
		Type: api.SecretTypeTLS,
	}

	_, err := NewACMECertDataFromSecret(certificateSecret)
	assert.NotNil(t, err)
	assert.Equal(t, "INTERNAL:Could not find key tls.crt in secret " + defaultCertPrefix + "hello", err.Error())

}

func TestClient(t *testing.T) {
	keyBits := 32 // small value keeps test fast
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		t.Fatal("Could not generate test key:", err)
	}
	user := &ACMEUserData{
		Email:        "test@test.com",
		Registration: new(acme.RegistrationResource),
		Key: pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}),
	}

	config := &ACMEConfig{
		Provider: "http",
		UserData: user,
	}
	_, err = NewACMEClient(config)
	assert.Nil(t, err)
}
