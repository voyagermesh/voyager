## Certificates
Voyager manages certificates objects to create Certificates default from Let's Encrypt.

### Core features of AppsCode Certificates:
  - Creates and stores Certificate from Let's Encrypt ot any custom provider supports ACME protocol
  - Auto renew certificate before expiration
  - Uses HTTP provider to issue certificate with HTTP request
  - Domain validation using ACME dns-01 challenges.
  - Support for multiple DNS providers.

### Object Expansion
Certificate Object is Defined as follows:

```go
type Certificate struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 CertificateSpec   `json:"spec,omitempty"`
	Status               CertificateStatus `json:"status,omitempty"`
}

type CertificateSpec struct {
	// Tries to obtain a single certificate using all domains passed into Domains.
	// The first domain in domains is used for the CommonName field of the certificate, all other
	// domains are added using the Subject Alternate Names extension.
	Domains []string `json:"domains,omitempty"`

	// DNS Provider.
	Provider string `json:"provider,omitempty"`
	Email    string `json:"email,omitempty"`

	// This is the ingress Reference that will be used if provider is http
	HTTPProviderIngressReference api.ObjectReference `json:"httpProviderIngressReference,omitempty"`

	// ProviderCredentialSecretName is used to create the acme client, that will do
	// needed processing in DNS.
	ProviderCredentialSecretName string `json:"providerCredentialSecretName,omitempty"`

	// Secret contains ACMEUser information. If empty tries to find an Secret via domains
	// if not found create an ACMEUser and stores as a secret.
	ACMEUserSecretName string `json:"acmeUserSecretName"`

	// ACME server that will be used to obtain this certificate.
	ACMEServerURL string `json:"acmeStagingURL"`
}

type CertificateStatus struct {
	CertificateObtained bool                   `json:"certificateObtained"`
	Message             string                 `json:"message"`
	Created             time.Time              `json:"created,omitempty"`
	ACMEUserSecretName  string                 `json:"acmeUserSecretName,omitempty"`
	Details             ACMECertificateDetails `json:"details,omitempty"`
}
```

### Explanation
  - apiVersion - The Kubernetes API version. See Certificate Third Party Resource.
  - kind - The Kubernetes object type.
  - metadata.name - The name of the Certificate object.
  - spec.domains - The DNS domains to obtain a for. First on the list will be the Common Name for the certificate.
  - spec.email - The email address used for registration.
  - spec.provider - The name of the dns provider plugin.
  - spec.providerCredentialSecretName - The Kubernetes secret that holds dns provider configuration.
  - spec.acmeUserSecretName - acme user information to use for obtaining certificates. If none is provided one will be created
  - spec.acmeStagingURL - server to obtain a certificate from. Default uses Let's Encrypt.

### Supported Providers
[This Providers](provider.md) are supported as domain's DNS provider. The `providerCredentialSecretName` Must match the
format.

## Usage
- [Creating a Certificate](create.md)
- [Deleting a Certificate](delete.md)
- [Consuming Certificates](consume.md)

## Using Certificate with Ingress
For sakes of simply managing ingress with TLS termination we can create a ingress with some Annotation that can be used
to create and or manage a certificate resource with Voyager controller. Read More with [Ingress](../ingress/tls.md)

```
certificate.appscode.com/enabled         // Enable certifiacte with ingress
certificate.appscode.com/name            // Name of the certificate
certificate.appscode.com/provider        // Name of the DNS provider
certificate.appscode.com/email           // Email address to use for registration
certificate.appscode.com/provider-secret // DNS provider secrets to manage DNS
```