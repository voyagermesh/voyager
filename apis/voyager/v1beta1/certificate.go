package v1beta1

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindCertificate = "Certificate"
	ResourceNameCertificate = "certificate"
	ResourceTypeCertificate = "certificates"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Certificate struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CertificateSpec   `json:"spec,omitempty"`
	Status            CertificateStatus `json:"status,omitempty"`
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
	HTTPProviderIngressReference LocalTypedReference `json:"httpProviderIngressReference,omitempty"`

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
	Ingress LocalTypedReference `json:"ingress,omitempty"`
}

type DNSChallengeProvider struct {
	// DNS Provider from the list https://github.com/appscode/voyager/blob/master/docs/user-guide/certificate/provider.md
	Provider             string `json:"provider,omitempty"`
	CredentialSecretName string `json:"credentialSecretName,omitempty"`
}

type CertificateStorage struct {
	Secret *apiv1.LocalObjectReference `json:"secret,omitempty"`
	Vault  *VaultStore                 `json:"vault,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items,omitempty"`
}
