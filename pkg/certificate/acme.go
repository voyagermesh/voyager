package certificate

import (
	"crypto"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/pkg/errors"
	"github.com/xenolf/lego/acme"
	"github.com/xenolf/lego/providers/dns/azure"
	"github.com/xenolf/lego/providers/dns/cloudflare"
	"github.com/xenolf/lego/providers/dns/digitalocean"
	"github.com/xenolf/lego/providers/dns/dnsimple"
	"github.com/xenolf/lego/providers/dns/dnsmadeeasy"
	"github.com/xenolf/lego/providers/dns/dyn"
	"github.com/xenolf/lego/providers/dns/fastdns"
	"github.com/xenolf/lego/providers/dns/gandi"
	"github.com/xenolf/lego/providers/dns/gcloud"
	"github.com/xenolf/lego/providers/dns/godaddy"
	"github.com/xenolf/lego/providers/dns/linode"
	"github.com/xenolf/lego/providers/dns/namecheap"
	"github.com/xenolf/lego/providers/dns/ovh"
	"github.com/xenolf/lego/providers/dns/pdns"
	"github.com/xenolf/lego/providers/dns/route53"
	"github.com/xenolf/lego/providers/dns/vultr"
)

const (
	LetsEncryptStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptProdURL    = "https://acme-v02.api.letsencrypt.org/directory"
)

func (c *Controller) newACMEClient() (*acme.Client, error) {
	client, err := acme.NewClient(c.acmeUser.getServerURL(), c.acmeUser, acme.RSA2048)
	if err != nil {
		return nil, err
	}

	newDNSProvider := func(provider acme.ChallengeProvider, err error) (*acme.Client, error) {
		if err != nil {
			return nil, err
		}
		if err := client.SetChallengeProvider(acme.DNS01, provider); err != nil {
			return nil, err
		}
		client.ExcludeChallenges([]acme.Challenge{acme.HTTP01})
		return client, nil
	}

	var found bool
	dnsLoader := func(key string) (value string, found bool) {
		v, found := c.DNSCredentials[key]
		return string(v), found
	}

	switch strings.ToLower(c.ChallengeProvider) {
	case "http":
		if err := client.SetChallengeProvider(acme.HTTP01, providers.DefaultHTTPProvider()); err != nil {
			return nil, err
		}
		client.ExcludeChallenges([]acme.Challenge{acme.DNS01})
		return client, nil
	case "aws", "route53":
		if c.cfg.CloudProvider == "aws" && len(c.DNSCredentials) == 0 {
			return newDNSProvider(route53.NewDNSProvider())
		}
		var accessKeyId, secretAccessKey, hostedZoneID string
		if accessKeyId, found = dnsLoader("AWS_ACCESS_KEY_ID"); !found {
			return nil, errors.Errorf("dns provider credential lacks required key %s", "AWS_ACCESS_KEY_ID")
		}
		if secretAccessKey, found = dnsLoader("AWS_SECRET_ACCESS_KEY"); !found {
			return nil, errors.Errorf("dns provider credential lacks required key %s", "AWS_SECRET_ACCESS_KEY")
		}
		// AWS_HOSTED_ZONE_ID is optional
		// If AWS_HOSTED_ZONE_ID is not set, Lego tries to determine the correct public hosted zone via the FQDN.
		// ref: https://github.com/xenolf/lego/blob/5a2fd5039fbba3c06b640be91a2c436bc23f74e8/providers/dns/route53/route53.go#L63
		if zoneID, found := dnsLoader("AWS_HOSTED_ZONE_ID"); found {
			hostedZoneID = zoneID
		}
		return newDNSProvider(route53.NewDNSProviderCredentials(accessKeyId, secretAccessKey, hostedZoneID))
	case "azure", "acs", "aks":
		var clientId, clientSecret, subscriptionId, tenantId, resourceGroup string
		if clientId, found = dnsLoader("AZURE_CLIENT_ID"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AZURE_CLIENT_ID")
		}
		if clientSecret, found = dnsLoader("AZURE_CLIENT_SECRET"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AZURE_CLIENT_SECRET")
		}
		if subscriptionId, found = dnsLoader("AZURE_SUBSCRIPTION_ID"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AZURE_SUBSCRIPTION_ID")
		}
		if tenantId, found = dnsLoader("AZURE_TENANT_ID"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AZURE_TENANT_ID")
		}
		if resourceGroup, found = dnsLoader("AZURE_RESOURCE_GROUP"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AZURE_RESOURCE_GROUP")
		}
		return newDNSProvider(azure.NewDNSProviderCredentials(clientId, clientSecret, subscriptionId, tenantId, resourceGroup))
	case "cloudflare":
		var email, key string
		if email, found = dnsLoader("CLOUDFLARE_EMAIL"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "CLOUDFLARE_EMAIL")
		}
		if key, found = dnsLoader("CLOUDFLARE_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "CLOUDFLARE_API_KEY")
		}
		return newDNSProvider(cloudflare.NewDNSProviderCredentials(email, key))
	case "digitalocean", "do":
		var apiAuthToken string
		if apiAuthToken, found = dnsLoader("DO_AUTH_TOKEN"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DO_AUTH_TOKEN")
		}
		return newDNSProvider(digitalocean.NewDNSProviderCredentials(apiAuthToken))
	case "dnsimple":
		var accessToken, baseUrl string
		if accessToken, found = dnsLoader("DNSIMPLE_OAUTH_TOKEN"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DNSIMPLE_OAUTH_TOKEN")
		}
		baseUrl, _ = dnsLoader("DNSIMPLE_BASE_URL")
		return newDNSProvider(dnsimple.NewDNSProviderCredentials(accessToken, baseUrl))
	case "dnsmadeeasy":
		var dnsmadeeasyAPIKey, dnsmadeeasyAPISecret string
		if dnsmadeeasyAPIKey, found = dnsLoader("DNSMADEEASY_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DNSMADEEASY_API_KEY")
		}
		if dnsmadeeasyAPISecret, found = dnsLoader("DNSMADEEASY_API_SECRET"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DNSMADEEASY_API_SECRET")
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
			return nil, errors.Errorf("dns provider credential missing key %s", "DYN_CUSTOMER_NAME")
		}
		if userName, found = dnsLoader("DYN_USER_NAME"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DYN_USER_NAME")
		}
		if password, found = dnsLoader("DYN_PASSWORD"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "DYN_PASSWORD")
		}
		return newDNSProvider(dyn.NewDNSProviderCredentials(customerName, userName, password))
	case "fastdns":
		var host, clientToken, clientSecret, accessToken string
		if host, found = dnsLoader("AKAMAI_HOST"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AKAMAI_HOST")
		}
		if clientToken, found = dnsLoader("AKAMAI_CLIENT_TOKEN"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AKAMAI_CLIENT_TOKEN")
		}
		if clientSecret, found = dnsLoader("AKAMAI_CLIENT_SECRET"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AKAMAI_CLIENT_SECRET")
		}
		if accessToken, found = dnsLoader("AKAMAI_ACCESS_TOKEN"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "AKAMAI_ACCESS_TOKEN")
		}
		return newDNSProvider(fastdns.NewDNSProviderClient(host, clientToken, clientSecret, accessToken))
	case "gandi":
		var apiKey string
		if apiKey, found = dnsLoader("GANDI_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "GANDI_API_KEY")
		}
		return newDNSProvider(gandi.NewDNSProviderCredentials(apiKey))
	case "godaddy":
		var apiKey, apiSecret string
		if apiKey, found = dnsLoader("GODADDY_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "GODADDY_API_KEY")
		}
		if apiSecret, found = dnsLoader("GODADDY_API_SECRET"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "GODADDY_API_SECRET")
		}
		return newDNSProvider(godaddy.NewDNSProviderCredentials(apiKey, apiSecret))
	case "googlecloud", "google", "gce", "gke":
		if (c.cfg.CloudProvider == "gce" || c.cfg.CloudProvider == "gke") && len(c.DNSCredentials) == 0 {
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
			return newDNSProvider(gcloud.NewDNSProviderCredentials(string(projectID), nil))
		}
		var project, jsonKey string
		if project, found = dnsLoader("GCE_PROJECT"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "GCE_PROJECT")
		}
		if sa, found := dnsLoader("GOOGLE_SERVICE_ACCOUNT_JSON_KEY"); found {
			jsonKey = sa
		} else {
			if jsonKey, found = dnsLoader("GCE_SERVICE_ACCOUNT_DATA"); !found { // deprecated key
				return nil, errors.Errorf("dns provider credential missing key %s", "GOOGLE_SERVICE_ACCOUNT_JSON_KEY")
			}
		}
		if len(jsonKey) <= 0 {
			return nil, errors.New("GCE_SERVICE_ACCOUNT_DATA is missing")
		}
		return newDNSProvider(gcloud.NewDNSProviderCredentials(string(project), []byte(jsonKey)))
	case "linode":
		var apiKey string
		if apiKey, found = dnsLoader("LINODE_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "LINODE_API_KEY")
		}
		return newDNSProvider(linode.NewDNSProviderCredentials(apiKey))
	case "namecheap":
		var apiUser, apiKey string
		if apiUser, found = dnsLoader("NAMECHEAP_API_USER"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "NAMECHEAP_API_USER")
		}
		if apiKey, found = dnsLoader("NAMECHEAP_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "NAMECHEAP_API_KEY")
		}
		return newDNSProvider(namecheap.NewDNSProviderCredentials(apiUser, apiKey))
	case "ovh":
		var apiEndpoint, applicationKey, applicationSecret, consumerKey string
		if apiEndpoint, found = dnsLoader("OVH_ENDPOINT"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "OVH_ENDPOINT")
		}
		if applicationKey, found = dnsLoader("OVH_APPLICATION_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "OVH_APPLICATION_KEY")
		}
		if applicationSecret, found = dnsLoader("OVH_APPLICATION_SECRET"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "OVH_APPLICATION_SECRET")
		}
		if consumerKey, found = dnsLoader("OVH_CONSUMER_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "OVH_CONSUMER_KEY")
		}
		return newDNSProvider(ovh.NewDNSProviderCredentials(apiEndpoint, applicationKey, applicationSecret, consumerKey))
	case "pdns":
		var key, apiURL string
		if key, found = dnsLoader("PDNS_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "PDNS_API_KEY")
		}
		if apiURL, found = dnsLoader("PDNS_API_URL"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "PDNS_API_URL")
		}
		hostUrl, err := url.Parse(apiURL)
		if err != nil {
			return nil, err
		}
		return newDNSProvider(pdns.NewDNSProviderCredentials(hostUrl, key))
	case "vultr":
		var apiKey string
		if apiKey, found = dnsLoader("VULTR_API_KEY"); !found {
			return nil, errors.Errorf("dns provider credential missing key %s", "VULTR_API_KEY")
		}
		return newDNSProvider(vultr.NewDNSProviderCredentials(apiKey))
	default:
		return nil, errors.New("Unknown provider specified")
	}
}

type ACMEUser struct {
	ServerURL    string
	Email        string
	Registration *acme.RegistrationResource
	Key          crypto.PrivateKey
}

var _ acme.User = &ACMEUser{}

func (u *ACMEUser) GetEmail() string {
	return u.Email
}

func (u *ACMEUser) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}

func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

func (u *ACMEUser) getServerURL() string {
	return u.ServerURL
}
