package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	ACMEUserEmail    = "ACME_EMAIL"
	ACMEUserDataJSON = "ACME_USER_DATA"
	ACMEServerURL    = "ACME_SERVER_URL"
)

func (c *Certificate) Migrate() (bool, *Certificate, *apiv1.Secret) {
	required := false
	if c.Spec.Provider == "http" {
		c.Spec.ChallengeProvider.HTTP = &HTTPChallengeProvider{
			Ingress: c.Spec.HTTPProviderIngressReference,
		}
		required = true
	} else if len(c.Spec.Provider) > 0 {
		c.Spec.ChallengeProvider.DNS = &DNSChallengeProvider{
			ProviderType:         c.Spec.Provider,
			CredentialSecretName: c.Spec.ProviderCredentialSecretName,
		}
		required = true
	}
	if c.Spec.Storage.Kubernetes == nil && c.Spec.Storage.Vault == nil {
		required = true
		c.Spec.Storage = CertificateStorage{Kubernetes: &CertificateStorageKubernetes{}}
	}

	if c.Spec.ACMEUserSecretName == "" {
		c.Spec.ACMEUserSecretName = "acme-" + c.Name
		required = true
	}

	var secretRequired *apiv1.Secret
	if len(c.Spec.Email) != 0 {
		secretRequired = &apiv1.Secret{
			ObjectMeta: v1.ObjectMeta{Name: c.Spec.ACMEUserSecretName, Namespace: c.Namespace},
			Data: map[string][]byte{
				ACMEUserEmail: []byte(c.Spec.Email),
			},
		}
		if len(c.Spec.ACMEServerURL) != 0 {
			secretRequired.Data[ACMEServerURL] = []byte(c.Spec.ACMEServerURL)
		}
		required = true
	}

	// Setting deprecated values to empty
	c.Spec.Provider = ""
	c.Spec.ProviderCredentialSecretName = ""
	c.Spec.Email = ""
	c.Spec.ACMEServerURL = ""

	return required, c, secretRequired
}
