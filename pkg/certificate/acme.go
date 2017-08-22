package certificate

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"strings"
	"sync"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/xenolf/lego/acme"
	"github.com/xenolf/lego/providers/dns/cloudflare"
	"github.com/xenolf/lego/providers/dns/digitalocean"
	"github.com/xenolf/lego/providers/dns/dnsimple"
	"github.com/xenolf/lego/providers/dns/dnsmadeeasy"
	"github.com/xenolf/lego/providers/dns/dyn"
	"github.com/xenolf/lego/providers/dns/gandi"
	"github.com/xenolf/lego/providers/dns/googlecloud"
	"github.com/xenolf/lego/providers/dns/linode"
	"github.com/xenolf/lego/providers/dns/namecheap"
	"github.com/xenolf/lego/providers/dns/ovh"
	"github.com/xenolf/lego/providers/dns/pdns"
	"github.com/xenolf/lego/providers/dns/rfc2136"
	"github.com/xenolf/lego/providers/dns/route53"
	"github.com/xenolf/lego/providers/dns/vultr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	certificateKey        = "certificate.appscode.com"
	LetsEncryptStagingURL = "https://acme-staging.api.letsencrypt.org/directory"
	LetsEncryptProdURL    = "https://acme-v01.api.letsencrypt.org/directory"
)

type ACMEClient struct {
	*acme.Client
	mu sync.Mutex

	HTTPProviderLock sync.Mutex
}

func NewACMEClient(config *ACMEConfig) (*ACMEClient, error) {
	providerUrl := LetsEncryptProdURL
	if config.ACMEServerUrl != "" {
		providerUrl = config.ACMEServerUrl
	}

	client, err := acme.NewClient(providerUrl, config.UserData, acme.RSA2048)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}

	initDNSProvider := func(provider acme.ChallengeProvider, err error) (*ACMEClient, error) {
		if err != nil {
			return nil, errors.FromErr(err).Err()
		}

		if err := client.SetChallengeProvider(acme.DNS01, provider); err != nil {
			return nil, errors.FromErr(err).Err()
		}

		client.ExcludeChallenges([]acme.Challenge{acme.HTTP01, acme.TLSSNI01})
		return &ACMEClient{
			Client: client,
		}, nil
	}

	keys := make([]string, 0)
	for k, v := range config.ProviderCredentials {
		replacer := strings.NewReplacer(
			"-", "_",
			" ", "_",
		)
		envKey := replacer.Replace(strings.ToUpper(k))
		os.Setenv(envKey, string(v))
		keys = append(keys, envKey)
	}

	defer func() {
		for _, key := range keys {
			os.Unsetenv(key)
		}
	}()

	switch config.Provider {
	case "http":
		defaultProvider := providers.DefaultHTTPProvider()
		if err := client.SetChallengeProvider(
			acme.HTTP01,
			defaultProvider,
		); err != nil {
			return nil, errors.FromErr(err).Err()
		}
		client.ExcludeChallenges([]acme.Challenge{acme.DNS01, acme.TLSSNI01})
		return &ACMEClient{
			Client: client,
		}, nil
	case "cloudflare":
		return initDNSProvider(cloudflare.NewDNSProvider())
	case "digitalocean":
		return initDNSProvider(digitalocean.NewDNSProvider())
	case "dnsimple":
		return initDNSProvider(dnsimple.NewDNSProvider())
	case "dnsmadeeasy":
		return initDNSProvider(dnsmadeeasy.NewDNSProvider())
	case "dyn":
		return initDNSProvider(dyn.NewDNSProvider())
	case "gandi":
		return initDNSProvider(gandi.NewDNSProvider())
	case "googlecloud":
		projectName := config.ProviderCredentials["GCE_PROJECT"]
		serviceAccountData := config.ProviderCredentials["GCE_SERVICE_ACCOUNT_DATA"]
		if len(serviceAccountData) <= 0 {
			return nil, errors.New("GCE_SERVICE_ACCOUNT_DATA is missing").Err()
		}
		return initDNSProvider(googlecloud.NewDNSProviderCredentials(string(projectName), serviceAccountData))
	case "linode":
		return initDNSProvider(linode.NewDNSProvider())
	case "namecheap":
		return initDNSProvider(namecheap.NewDNSProvider())
	case "ovh":
		return initDNSProvider(ovh.NewDNSProvider())
	case "pdns":
		return initDNSProvider(pdns.NewDNSProvider())
	case "rfc2136":
		return initDNSProvider(rfc2136.NewDNSProvider())
	case "aws", "route53":
		return initDNSProvider(route53.NewDNSProvider())
	case "vultr":
		return initDNSProvider(vultr.NewDNSProvider())
	default:
		return nil, errors.New("Unknown provider specified").Err()
	}
}

type ACMEConfig struct {
	Provider            string
	ACMEServerUrl       string
	ProviderCredentials map[string][]byte
	UserData            *ACMEUserData
	UserDataMap         map[string][]byte
}

type ACMEUserData struct {
	Email        string                     `json:"email"`
	Registration *acme.RegistrationResource `json:"registration"`
	Key          []byte                     `json:"key"`
}

type ACMECertDetails struct {
	Domain        []string `json:"domains"`
	CertURL       string   `json:"certUrl"`
	CertStableURL string   `json:"certStableUrl"`
	AccountRef    string   `json:"accountRef,omitempty"`
}

func (u *ACMEUserData) GetEmail() string {
	return u.Email
}

func (u *ACMEUserData) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}

func (u *ACMEUserData) GetPrivateKey() crypto.PrivateKey {
	pemBlock, _ := pem.Decode(u.Key)
	if pemBlock.Type != "RSA PRIVATE KEY" {
		log.Infof("Invalid PEM user key: Expected RSA PRIVATE KEY, got %v", pemBlock.Type)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		log.Infof("Error while parsing private key: %v", err)
	}

	return privateKey
}

func (u *ACMEUserData) Json() []byte {
	b, err := json.Marshal(u)
	if err != nil {
		return []byte("{}")
	}
	return b
}

type DomainCollection []string

func NewDomainCollection(domain ...string) *DomainCollection {
	d := &DomainCollection{}
	*d = append(*d, domain...)
	return d
}

func (d *DomainCollection) Append(domain ...string) *DomainCollection {
	*d = append(*d, domain...)
	return d
}

func (d *DomainCollection) String() string {
	val, _ := json.Marshal(d)
	return string(val)
}

func (d *DomainCollection) StringSlice() []string {
	return []string(*d)
}

func (d *DomainCollection) FromString(data string) *DomainCollection {
	d = &DomainCollection{}
	json.Unmarshal([]byte(data), d)
	return d
}

type ACMECertData struct {
	Domains    *DomainCollection
	Cert       []byte
	PrivateKey []byte
}

func NewACMECertDataFromSecret(s *apiv1.Secret, cert *api.Certificate) (ACMECertData, error) {
	var acmeCertData ACMECertData
	var ok bool
	acmeCertData.Domains = NewDomainCollection(cert.Spec.Domains...)
	acmeCertData.Cert, ok = s.Data[apiv1.TLSCertKey]
	if !ok {
		return acmeCertData, errors.New().WithMessagef("Could not find key tls.crt in secret %v", s.Name).Err()
	}
	acmeCertData.PrivateKey, ok = s.Data[apiv1.TLSPrivateKeyKey]
	if !ok {
		return acmeCertData, errors.New().WithMessagef("Could not find key tls.key in secret %v", s.Name).Err()
	}
	return acmeCertData, nil
}

func (c *ACMECertData) ToSecret(name, namespace string) *apiv1.Secret {
	log.Infoln("Revived certificates for name", name, "namespace", namespace)
	data := make(map[string][]byte)

	if len(c.Cert) > 0 {
		data[apiv1.TLSCertKey] = c.Cert
	}
	if len(c.PrivateKey) > 0 {
		data[apiv1.TLSPrivateKeyKey] = c.PrivateKey
	}
	log.Infoln("Certificate cert length", len(c.Cert), "private key length", len(c.PrivateKey))
	return &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "Secret",
		},
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCertPrefix + name,
			Namespace: namespace,
			Labels: map[string]string{
				certificateKey: "true",
			},
			Annotations: map[string]string{
				certificateKey: "true",
			},
		},
		Type: apiv1.SecretTypeTLS,
	}
}

func (a ACMECertData) EqualDomains(c *x509.Certificate) bool {
	certDomains := sets.NewString(c.Subject.CommonName)
	certDomains.Insert(c.DNSNames...)

	acmeDomains := sets.NewString(a.Domains.StringSlice()...)
	return certDomains.Equal(acmeDomains)
}
