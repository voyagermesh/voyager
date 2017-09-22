package certificate

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/xenolf/lego/acme"
	"github.com/xenolf/lego/providers/dns/azure"
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
	"github.com/xenolf/lego/providers/dns/route53"
	"github.com/xenolf/lego/providers/dns/vultr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	LetsEncryptStagingURL = "https://acme-staging.api.letsencrypt.org/directory"
	LetsEncryptProdURL    = "https://acme-v01.api.letsencrypt.org/directory"
)

type ACMEClient struct {
	*acme.Client
	mu sync.Mutex

	HTTPProviderLock sync.Mutex
}

func NewACMEClient(cfg *ACMEConfig) (*ACMEClient, error) {
	client, err := acme.NewClient(cfg.UserSecret.GetServerURL(), cfg.UserData, acme.RSA2048)
	if err != nil {
		return nil, err
	}

	newDNSProvider := func(provider acme.ChallengeProvider, err error) (*ACMEClient, error) {
		if err != nil {
			return nil, err
		}
		if err := client.SetChallengeProvider(acme.DNS01, provider); err != nil {
			return nil, err
		}
		client.ExcludeChallenges([]acme.Challenge{acme.HTTP01, acme.TLSSNI01})
		return &ACMEClient{Client: client}, nil
	}

	var found bool
	dnsLoader := func(key string) (value string, found bool) {
		v, found := cfg.DNSCredentials[key]
		return string(v), found
	}

	switch strings.ToLower(cfg.ChallengeProvider) {
	case "http":
		if err := client.SetChallengeProvider(acme.HTTP01, providers.DefaultHTTPProvider()); err != nil {
			return nil, err
		}
		client.ExcludeChallenges([]acme.Challenge{acme.DNS01, acme.TLSSNI01})
		return &ACMEClient{
			Client: client,
		}, nil
	case "aws", "route53":
		if cfg.CloudProvider == "aws" && len(cfg.DNSCredentials) == 0 {
			return newDNSProvider(route53.NewDNSProvider())
		}
		var accessKeyId, secretAccessKey string
		if accessKeyId, found = dnsLoader("AWS_ACCESS_KEY_ID"); !found {
			return nil, fmt.Errorf("dns provider credential lacks required key %s", "AWS_ACCESS_KEY_ID")
		}
		if secretAccessKey, found = dnsLoader("AWS_SECRET_ACCESS_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential lacks required key %s", "AWS_SECRET_ACCESS_KEY")
			return nil, fmt.Errorf("dns provider credential missing key %s", "")
		}
		return newDNSProvider(route53.NewDNSProviderCredentials(accessKeyId, secretAccessKey))
	case "azure", "acs":
		var clientId, clientSecret, subscriptionId, tenantId, resourceGroup string
		if clientId, found = dnsLoader("AZURE_CLIENT_ID"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "AZURE_CLIENT_ID")
		}
		if clientSecret, found = dnsLoader("AZURE_CLIENT_SECRET"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "AZURE_CLIENT_SECRET")
		}
		if subscriptionId, found = dnsLoader("AZURE_SUBSCRIPTION_ID"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "AZURE_SUBSCRIPTION_ID")
		}
		if tenantId, found = dnsLoader("AZURE_TENANT_ID"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "AZURE_TENANT_ID")
		}
		if resourceGroup, found = dnsLoader("AZURE_RESOURCE_GROUP"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "AZURE_RESOURCE_GROUP")
		}
		return newDNSProvider(azure.NewDNSProviderCredentials(clientId, clientSecret, subscriptionId, tenantId, resourceGroup))
	case "cloudflare":
		var email, key string
		if email, found = dnsLoader("CLOUDFLARE_EMAIL"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "CLOUDFLARE_EMAIL")
		}
		if key, found = dnsLoader("CLOUDFLARE_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "CLOUDFLARE_API_KEY")
		}
		return newDNSProvider(cloudflare.NewDNSProviderCredentials(email, key))
	case "digitalocean", "do":
		var apiAuthToken string
		if apiAuthToken, found = dnsLoader("DO_AUTH_TOKEN"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DO_AUTH_TOKEN")
		}
		return newDNSProvider(digitalocean.NewDNSProviderCredentials(apiAuthToken))
	case "dnsimple":
		var accessToken, baseUrl string
		if accessToken, found = dnsLoader("DNSIMPLE_OAUTH_TOKEN"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DNSIMPLE_OAUTH_TOKEN")
		}
		if baseUrl, found = dnsLoader("DNSIMPLE_BASE_URL"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DNSIMPLE_BASE_URL")
		}
		return newDNSProvider(dnsimple.NewDNSProviderCredentials(accessToken, baseUrl))
	case "dnsmadeeasy":
		var dnsmadeeasyAPIKey, dnsmadeeasyAPISecret string
		if dnsmadeeasyAPIKey, found = dnsLoader("DNSMADEEASY_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DNSMADEEASY_API_KEY")
		}
		if dnsmadeeasyAPISecret, found = dnsLoader("DNSMADEEASY_API_SECRET"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DNSMADEEASY_API_SECRET")
		}
		var baseURL string
		dnsmadeeasySandbox, _ := dnsLoader("DNSMADEEASY_SANDBOX")
		sandbox, _ := strconv.ParseBool(dnsmadeeasySandbox)
		if sandbox {
			baseURL = "https://api.sandbox.dnsmadeeasy.com/V2.0"
		} else {
			baseURL = "https://api.dnsmadeeasy.com/V2.0"
		}
		return newDNSProvider(dnsmadeeasy.NewDNSProviderCredentials(baseURL, dnsmadeeasyAPIKey, dnsmadeeasyAPISecret))
	case "dyn":
		var customerName, userName, password string
		if customerName, found = dnsLoader("DYN_CUSTOMER_NAME"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DYN_CUSTOMER_NAME")
		}
		if userName, found = dnsLoader("DYN_USER_NAME"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DYN_USER_NAME")
		}
		if password, found = dnsLoader("DYN_PASSWORD"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "DYN_PASSWORD")
		}
		return newDNSProvider(dyn.NewDNSProviderCredentials(customerName, userName, password))
	case "gandi":
		var apiKey string
		if apiKey, found = dnsLoader("GANDI_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "GANDI_API_KEY")
		}
		return newDNSProvider(gandi.NewDNSProviderCredentials(apiKey))
	case "googlecloud", "google", "gce", "gke":
		if (cfg.CloudProvider == "gce" || cfg.CloudProvider == "gke") && len(cfg.DNSCredentials) == 0 {
			// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
			// curl "http://metadata.google.internal/computeMetadata/v1/project/project-id" -H "Metadata-Flavor: Google"
			req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Metadata-Flavor", "Google")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			projectID, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return newDNSProvider(googlecloud.NewDNSProviderCredentials(string(projectID), nil))
		}
		var project, jsonKey string
		if project, found = dnsLoader("GCE_PROJECT"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "GCE_PROJECT")
		}
		if sa, found := dnsLoader("GOOGLE_SERVICE_ACCOUNT_JSON_KEY"); found {
			jsonKey = sa
		} else {
			if jsonKey, found = dnsLoader("GCE_SERVICE_ACCOUNT_DATA"); !found { // deprecated key
				return nil, fmt.Errorf("dns provider credential missing key %s", "GOOGLE_SERVICE_ACCOUNT_JSON_KEY")
			}
		}
		if len(jsonKey) <= 0 {
			return nil, errors.New("GCE_SERVICE_ACCOUNT_DATA is missing").Err()
		}
		return newDNSProvider(googlecloud.NewDNSProviderCredentials(string(project), []byte(jsonKey)))
	case "linode":
		var apiKey string
		if apiKey, found = dnsLoader("LINODE_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "LINODE_API_KEY")
		}
		return newDNSProvider(linode.NewDNSProviderCredentials(apiKey))
	case "namecheap":
		var apiUser, apiKey string
		if apiUser, found = dnsLoader("NAMECHEAP_API_USER"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "NAMECHEAP_API_USER")
		}
		if apiKey, found = dnsLoader("NAMECHEAP_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "NAMECHEAP_API_KEY")
		}
		return newDNSProvider(namecheap.NewDNSProviderCredentials(apiUser, apiKey))
	case "ovh":
		var apiEndpoint, applicationKey, applicationSecret, consumerKey string
		if apiEndpoint, found = dnsLoader("OVH_ENDPOINT"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "OVH_ENDPOINT")
		}
		if applicationKey, found = dnsLoader("OVH_APPLICATION_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "OVH_APPLICATION_KEY")
		}
		if applicationSecret, found = dnsLoader("OVH_APPLICATION_SECRET"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "OVH_APPLICATION_SECRET")
		}
		if consumerKey, found = dnsLoader("OVH_CONSUMER_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "OVH_CONSUMER_KEY")
		}
		return newDNSProvider(ovh.NewDNSProviderCredentials(apiEndpoint, applicationKey, applicationSecret, consumerKey))
	case "pdns":
		var key, apiURL string
		if key, found = dnsLoader("PDNS_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "PDNS_API_KEY")
		}
		if apiURL, found = dnsLoader("PDNS_API_URL"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "PDNS_API_URL")
		}
		hostUrl, err := url.Parse(apiURL)
		if err != nil {
			return nil, err
		}
		return newDNSProvider(pdns.NewDNSProviderCredentials(hostUrl, key))
	case "vultr":
		var apiKey string
		if apiKey, found = dnsLoader("VULTR_API_KEY"); !found {
			return nil, fmt.Errorf("dns provider credential missing key %s", "VULTR_API_KEY")
		}
		return newDNSProvider(vultr.NewDNSProviderCredentials(apiKey))
	default:
		return nil, errors.New("Unknown provider specified").Err()
	}
}

type ACMEUserSecret map[string][]byte

func (a ACMEUserSecret) GetEmail() string {
	return string(a[tapi.ACMEUserEmail])
}

func (a ACMEUserSecret) GetUserData() []byte {
	return a[tapi.ACMEUserDataJSON]
}

func (a ACMEUserSecret) GetServerURL() string {
	if u, found := a[tapi.ACMEServerURL]; found {
		return string(u)
	}
	return LetsEncryptProdURL
}

type ACMEConfig struct {
	CloudProvider     string
	ChallengeProvider string
	DNSCredentials    map[string][]byte
	UserData          *ACMEUserData
	UserDataMap       map[string][]byte
	UserSecret        ACMEUserSecret
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

func (a *ACMECertData) ToSecret(name, namespace, secretName string) *apiv1.Secret {
	log.Infoln("Revived certificates for name", name, "namespace", namespace)
	data := make(map[string][]byte)

	if len(a.Cert) > 0 {
		data[apiv1.TLSCertKey] = a.Cert
	}
	if len(a.PrivateKey) > 0 {
		data[apiv1.TLSPrivateKeyKey] = a.PrivateKey
	}
	log.Infoln("Certificate cert length", len(a.Cert), "private key length", len(a.PrivateKey))

	if len(secretName) == 0 {
		secretName = defaultCertPrefix + name
	}
	return &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "Secret",
		},
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
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
