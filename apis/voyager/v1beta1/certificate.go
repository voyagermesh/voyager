package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	ResourceKindCertificate = "Certificate"
	ResourceNameCertificate = "certificate"
	ResourceTypeCertificate = "certificates"
)

// +genclient=true
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
	//  email -> required, if not provided it will through error.
	//  acme-server-url -> custom server url to generate certificates, default is lets encrypt.
	//  acme-user-data -> user data, if not found one will be created for the provided email,
	//    and stored in the key.
	ACMEUserSecretName string `json:"acmeUserSecretName"`

	// Storage backend to store the certificates currently, kubernetes secret and vault.
	Storage CertificateStorage `json:"storage,omitempty"`
}

type ChallengeProvider struct {
	HTTP *HTTPChallengeProvider `json:"http,omitempty"`
	DNS  *DNSChallengeProvider  `json:"dns,omitempty"`
}

type HTTPChallengeProvider struct {
	Ingress apiv1.ObjectReference `json:"ingress,omitempty"`
}

type DNSChallengeProvider struct {
	ProviderType         string `json:"providerType,omitempty"`
	CredentialSecretName string `json:"credentialSecretName,omitempty"`
}

type CertificateStorage struct {
	Kubernetes *CertificateStorageKubernetes `json:"kubernetes,omitempty"`
	Vault      *CertificateStorageVault      `json:"vault,omitempty"`
}

type CertificateStorageKubernetes struct {
	// Secret name to store the certificate, default cert-<certificate-name>
	Name string `json:"name,omitempty"`
}

type CertificateStorageVault struct {
	Name string `json:"name,omitempty"`
	// Address is the address of the Vault server. This should be a complete
	// URL such as "http://vault.example.com:8082".
	Address string `json:"address,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	// Should be TokenSecretName? @tamal
	Token string `json:"token,omitempty"`
}

type CertificateStatus struct {
	CertificateObtained bool                   `json:"certificateObtained"`
	CreationTime        *metav1.Time           `json:"creationTime,omitempty"`
	Conditions          []CertificateCondition `json:"conditions,omitempty"`
	Certificate         ACMECertificateDetails `json:"certificate,omitempty"`
}

type ACMECertificateDetails struct {
	Domain        string `json:"domain"`
	CertURL       string `json:"certUrl"`
	CertStableURL string `json:"certStableUrl"`
	AccountRef    string `json:"accountRef,omitempty"`
}

type RequestConditionType string

// These are the possible conditions for a certificate create request.
const (
	CertificateCreated RequestConditionType = "Created"
	CertificateUpdated RequestConditionType = "Updated"
	CertificateFailed  RequestConditionType = "Failed"
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
