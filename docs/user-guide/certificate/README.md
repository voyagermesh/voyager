# Certificate

Voyager comes with a built-in certificate manager that can issue free TLS/SSL certificates from Let's Encrypt.


Voyager can automatically provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes Certificate resource.

Features

Provision free TLS certificates from Let's Encrypt,
Manage issued certificates using a Kubernetes Third Party Resource,
Domain validation using ACME dns-01 challenges,
Support for multiple DNS providers,
Auto Renew Certificates,
Use issued Certificates with Ingress to Secure Communications.


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

	// ChallengeProvider details to verify domains
	ChallengeProvider ChallengeProvider `json:"challengeProvider"`

	// Secret contains ACMEUser information. Secret must contain a key `email`
	// If empty tries to find an Secret via domains
	// if not found create an ACMEUser and stores as a secret.
	// Secrets key to be expected:
	//  ACME_EMAIL -> required, if not provided it will through error.
	//  ACME_SERVER_URL -> custom server url to generate certificates, default is lets encrypt.
	//  ACME_USER_DATA -> user data, if not found one will be created for the provided email,
	//    and stored in the key.
	ACMEUserSecretName string `json:"acmeUserSecretName"`

	// Storage backend to store the certificates currently, kubernetes secret and vault.
	Storage CertificateStorage `json:"storage,omitempty"`

	// Following fields are deprecated and will removed in future version.
	// https://github.com/appscode/voyager/pull/506
	// Deprecated. DNS Provider.
	Provider string `json:"provider,omitempty"`
	// Deprecated
	Email string `json:"email,omitempty"`

	// This is the ingress Reference that will be used if provider is http
	// Deprecated
	HTTPProviderIngressReference apiv1.ObjectReference `json:"httpProviderIngressReference,omitempty"`

	// ProviderCredentialSecretName is used to create the acme client, that will do
	// needed processing in DNS.
	// Deprecated
	ProviderCredentialSecretName string `json:"providerCredentialSecretName,omitempty"`

	// ACME server that will be used to obtain this certificate.
	// Deprecated
	ACMEServerURL string `json:"acmeStagingURL,omitempty"`
}

type ChallengeProvider struct {
	HTTP *HTTPChallengeProvider `json:"http,omitempty"`
	DNS  *DNSChallengeProvider  `json:"dns,omitempty"`
}

type HTTPChallengeProvider struct {
	Ingress apiv1.ObjectReference `json:"ingress,omitempty"`
}

type DNSChallengeProvider struct {
	// DNS Provider from the list https://github.com/appscode/voyager/blob/master/docs/user-guide/certificate/provider.md
	Provider             string `json:"provider,omitempty"`
	CredentialSecretName string `json:"credentialSecretName,omitempty"`
}

type CertificateStorage struct {
	Secret *SecretStore `json:"secret,omitempty"`
	Vault  *VaultStore  `json:"vault,omitempty"`
}

type SecretStore struct {
	// Secret name to store the certificate, default cert-<certificate-name>
	Name string `json:"name,omitempty"`
}

type VaultStore struct {
	Name   string `json:"name,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type CertificateStatus struct {
	CreationTime          *metav1.Time           `json:"creationTime,omitempty"`
	Conditions            []CertificateCondition `json:"conditions,omitempty"`
	LastIssuedCertificate *CertificateDetails    `json:"lastIssuedCertificate,omitempty"`
	// Deprecated
	CertificateObtained bool `json:"certificateObtained,omitempty"`
	// Deprecated
	Message string `json:"message, omitempty"`
	// Deprecated
	ACMEUserSecretName string `json:"acmeUserSecretName,omitempty"`
	// Deprecated
	Details *ACMECertificateDetails `json:"details,omitempty"`
}

type ACMECertificateDetails struct {
	Domain        string `json:"domain"`
	CertURL       string `json:"certUrl"`
	CertStableURL string `json:"certStableUrl"`
	AccountRef    string `json:"accountRef,omitempty"`
}

type CertificateDetails struct {
	SerialNumber  string      `json:"serialNumber,omitempty"`
	NotBefore     metav1.Time `json:"notBefore,omitempty"`
	NotAfter      metav1.Time `json:"notAfter,omitempty"`
	CertURL       string      `json:"certURL"`
	CertStableURL string      `json:"certStableURL"`
	AccountRef    string      `json:"accountRef,omitempty"`
}

type RequestConditionType string

// These are the possible conditions for a certificate create request.
const (
	CertificateIssued      RequestConditionType = "Issued"
	CertificateFailed      RequestConditionType = "Failed"
	CertificateRateLimited RequestConditionType = "RateLimited"
)

type CertificateCondition struct {
	// request approval state, currently Approved or Denied.
	Type RequestConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=RequestConditionType"`
	// brief reason for the request state
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,2,opt,name=reason"`
	// human readable message with details about the request state
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// timestamp for the last update to this condition
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,4,opt,name=lastUpdateTime"`
}
```

### Supported Providers
[This Providers](provider.md) are supported as domain's DNS provider. The `providerCredentialSecretName` Must match the
format.

## Usage
- [Creating a Certificate](create.md)
- [Creating a Certificate with custom provider](create-with-custom-provider.md)
- [Deleting a Certificate](delete.md)
- [Consuming Certificates](consume.md)

## Using Certificate with Ingress

For sakes of simply managing ingress with TLS termination we can create a ingress with some Annotation that can be used
to create and or manage a certificate resource with Voyager controller. Read More with [Ingress](../ingress/tls.md)

Read the example how to use [HTTP Provider](/docs/user-guide/certificate/create.md#create-certificate-with-http-provider)
for certificate.
