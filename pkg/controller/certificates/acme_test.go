package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"

	"crypto/x509"
	"encoding/pem"

	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/acme"
)

func TestNewDomainCollection(t *testing.T) {
	d := NewDomainCollection("a.com")
	assert.Equal(t, `["a.com"]`, d.String())
	fmt.Println(d.String())

	d.Append("hello.world").Append("foo.bar")
	assert.Equal(t, `["a.com","hello.world","foo.bar"]`, d.String())
	fmt.Println(d.String())
}

func TestACMECertData(t *testing.T) {

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
		Provider: "googlecloud",
		ProviderCredentials: map[string][]byte{
			"GCE_PROJECT": []byte("tigerworks-kube"),
		},
		UserData: user,
	}

	_, err = NewACMEClient(config)
	assert.Nil(t, err)
}
