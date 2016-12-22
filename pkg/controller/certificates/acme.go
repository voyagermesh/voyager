package certificates

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"strings"
	"sync"

	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/certificates/providers"
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
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

const (
	certificateKey     = "certificate.appscode.com"
	LetsEncryptACMEUrl = "https://acme-staging.api.letsencrypt.org/directory"
)

type ACMEClient struct {
	*acme.Client
	mu sync.Mutex

	HTTPProviderLock sync.Mutex
}

func NewACMEClient(config *ACMEConfig) (*ACMEClient, error) {
	providerUrl := LetsEncryptACMEUrl
	if config.ACMEServerUrl != "" {
		providerUrl = config.ACMEServerUrl
	}

	client, err := acme.NewClient(providerUrl, config.UserData, acme.RSA2048)
	if err != nil {
		return nil, errors.New().WithCause(err).Internal()
	}

	initDNSProvider := func(provider acme.ChallengeProvider, err error) (*ACMEClient, error) {
		if err != nil {
			return nil, errors.New().WithCause(err).Internal()
		}

		if err := client.SetChallengeProvider(acme.DNS01, provider); err != nil {
			return nil, errors.New().WithCause(err).Internal()
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
			return nil, errors.New().WithCause(err).Internal()
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
			return nil, errors.New().WithMessage("GCE_SERVICE_ACCOUNT_DATA is missing").Internal()
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
	case "route53":
		return initDNSProvider(route53.NewDNSProvider())
	case "vultr":
		return initDNSProvider(vultr.NewDNSProvider())
	default:
		return nil, errors.New().WithMessage("Unknown provider specified").NotFound()
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

func NewACMECertDataFromSecret(s *api.Secret) (ACMECertData, error) {
	var acmeCertData ACMECertData
	var ok bool

	acmeCertData.Domains = NewDomainCollection().FromString(s.Labels[certificateKey+"/domains"])
	acmeCertData.Cert, ok = s.Data[api.TLSCertKey]
	if !ok {
		return acmeCertData, errors.New().WithMessagef("Could not find key tls.crt in secret %v", s.Name).Internal()
	}
	acmeCertData.PrivateKey, ok = s.Data[api.TLSPrivateKeyKey]
	if !ok {
		return acmeCertData, errors.New().WithMessagef("Could not find key tls.key in secret %v", s.Name).Internal()
	}
	return acmeCertData, nil
}

func (c *ACMECertData) ToSecret(name, namespace string) *api.Secret {
	log.Infoln("Revived certificates for name", name, "namespace", namespace)
	data := make(map[string][]byte)

	if len(c.Cert) > 0 {
		data[api.TLSCertKey] = c.Cert
	}
	if len(c.PrivateKey) > 0 {
		data[api.TLSPrivateKeyKey] = c.PrivateKey
	}
	log.Infoln("Certificate cert length", len(c.Cert), "private key length", len(c.PrivateKey))
	return &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "api",
			Kind:       "Secret",
		},
		Data: data,
		ObjectMeta: api.ObjectMeta{
			Name:      defaultCertPrefix + name,
			Namespace: namespace,
			Labels: map[string]string{
				certificateKey:              "true",
				certificateKey + "/domains": c.Domains.String(),
			},
			Annotations: map[string]string{
				certificateKey: "true",
			},
		},
		Type: api.SecretTypeTLS,
	}
}

func (a ACMECertData) EqualDomains(c *x509.Certificate) bool {
	certDomains := []string{
		c.Subject.CommonName,
	}
	certDomains = append(certDomains, c.DNSNames...)
	return stringutil.EqualSlice(certDomains, a.Domains.StringSlice())
}
