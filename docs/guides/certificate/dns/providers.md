---
title: Supported DNS Challenge Providers
description: Supported DNS Challenge Providers
menu:
  product_voyager_6.0.0:
    identifier: providers-dns
    name: Supported Providers
    parent: dns-certificate
    weight: 20
product_name: voyager
menu_name: product_voyager_6.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Supported DNS Challenge Providers

To issue SSL certificate using Let's Encrypt DNS-01 challenge, Voyager operator requires necessary permission to add and remove a TXT record for domain `_acme-challenge.<domain>` to complete the DNS challenge.

## Supported DNS providers
Please see the list of supported providers and the keys expected in credential provider secret.

### Amazon Route53
 - Provider: `aws` or `route53`
 - Credential secret keys:
   - `AWS_ACCESS_KEY_ID`: The access key id
   - `AWS_SECRET_ACCESS_KEY`: The secret corresponding to the access key
   - `AWS_HOSTED_ZONE_ID`: `Optional`. If AWS_HOSTED_ZONE_ID is not set, Voyager tries to determine the correct public hosted zone via the FQDN.

To learn about necessary IAM permissions, please see [here](/docs/guides/certificate/dns/route53.md).

### Microsoft Azure
 - Provider: `azure` or `acs`
 - Credential secret keys:
   - `AZURE_CLIENT_ID`: Azure client id
   - `AZURE_CLIENT_SECRET`: The secret corresponding to the client id
   - `AZURE_SUBSCRIPTION_ID`: Azure subscription id
   - `AZURE_TENANT_ID`: Azure tenant id
   - `AZURE_RESOURCE_GROUP`: Azure resource group where domain is hosted

### Cloudflare
 - Provider: `cloudflare`
 - Credential secret keys:
   - `CLOUDFLARE_EMAIL`: The email of a cloudflare user
   - `CLOUDFLARE_API_KEY`: The API key corresponding to the email

### Digital Ocean
 - Provider: `digitalocean` or `do`
 - Credential secret keys:
   - `DO_AUTH_TOKEN`: The write scoped api token for a DigitalOcean user

### DNSimple
 - Provider: `dnsimple`
 - Credential secret keys:
   - `DNSIMPLE_OAUTH_TOKEN`: The oauth token for a DNSimple user
   - `DNSIMPLE_BASE_URL`: `Optional`. The base url of DNSimple server

### DNS Made Easy
 - Provider: `dnsmadeeasy`
 - Credential secret keys:
   - `DNSMADEEASY_API_KEY`: The api key for a DNS Made Easy user
   - `DNSMADEEASY_API_SECRET`: The api secret corresponding with the api key
   - `DNSMADEEASY_SANDBOX`: `Optional`. A boolean flag, if set to `true` or `1`, requests will be sent to the sandbox API

### Dyn
 - Provider: `dyn`
 - Credential secret keys:
   - `DYN_CUSTOMER_NAME`: The customer name of a Dyn user
   - `DYN_USER_NAME`: The user name of the Dyn user
   - `DYN_PASSWORD`: The password of the Dyn user

### Gandi
 - Provider: `gandi`
 - Credential secret keys:
   - `GANDI_API_KEY`: The API key for a Gandi user

### GoDaddy
 - Provider: `godaddy`
 - Credential secret keys:
   - `GODADDY_API_KEY`: The API key for a GoDaddy user
   - `GODADDY_API_SECRET`: The api secret for the api key

### Google Cloud DNS
 - Provider: `googlecloud` or `google` or `gce` or `gke`
 - Credential secret keys:
   - `GCE_PROJECT`: The name of the Google Cloud project to use
   - `GOOGLE_SERVICE_ACCOUNT_JSON_KEY`: Service account json downloaded from Google Cloud console. This service account requires scope `https://www.googleapis.com/auth/ndev.clouddns.readwrite` to view and manage your DNS records hosted by Google Cloud DNS.

If you are running your cluster on Google Cloud (GKE or GCE), Voyager can use default service account associated with a VM. Please see [here](/docs/guides/certificate/dns/google-cloud.md) for detailed instructions.

### Linode
 - Provider: `linode`
 - Credential secret keys:
   - `LINODE_API_KEY`: The API key for a linode user.

### Namecheap
 - Provider: `namecheap`
 - Credential secret keys:
   - `NAMECHEAP_API_USER`: The username of a Namecheap user
   - `NAMECHEAP_API_KEY`: The API key corresponding with the Namecheap user

### OVH
 - Provider: `ovh`
 - Credential secret keys:
   - `OVH_ENDPOINT`: The URL of the API endpoint to use
   - `OVH_APPLICATION_KEY`: The application key
   - `OVH_APPLICATION_SECRET`: The secret corresponding to the application key
   - `OVH_CONSUMER_KEY`: The consumer key

### PDNS
 - Provider: `pdns`
 - Credential secret keys:
   - `PDNS_API_KEY`: The API key to use
   - `PDNS_API_URL`: PDNS api server address

### Vultr
 - Provider: `vultr`
 - Credential secret keys:
   - `VULTR_API_KEY`: The API key to use


## How to provide DNS provider credential

To provide DNS provider credential, create a secret with appropriate keys. Then pass the secret name to the `spec.challengeProvider.dns.credentialSecretName` field. Both the `Secret` and `Certificate` object must reside in the same namespace.

```console
# create secret for AWS route53
kubectl create secret generic voyager-route53 --namespace default \
  --from-literal=AWS_ACCESS_KEY_ID=INSERT_YOUR_ACCESS_KEY_ID_HERE \
  --from-literal=AWS_SECRET_ACCESS_KEY=INSERT_YOUR_SECRET_ACCESS_KEY_HERE \
  --from-literal=AWS_HOSTED_ZONE_ID=INSERT_YOUR_HOSTED_ZONE_ID_HERE

kubectl get secret voyager-route53 -o yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: SU5TRVJUX1lPVVJfQUNDRVNTX0tFWV9JRF9IRVJF
  AWS_HOSTED_ZONE_ID: SU5TRVJUX1lPVVJfSE9TVEVEX1pPTkVfSURfSEVSRQ==
  AWS_SECRET_ACCESS_KEY: SU5TRVJUX1lPVVJfU0VDUkVUX0FDQ0VTU19LRVlfSEVSRQ==
kind: Secret
metadata:
  creationTimestamp: 2017-11-27T23:17:31Z
  name: voyager-route53
  namespace: default
  resourceVersion: "16160"
  selfLink: /api/v1/namespaces/default/secrets/voyager-route53
  uid: 24949869-d3c9-11e7-98b3-08002787a1b5
type: Opaque
```

Here is an example `Certificate` CRD.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Certificate
metadata:
  name: kitecipro-iam
  namespace: default
spec:
  domains:
  - kiteci.pro
  - www.kiteci.pro
  acmeUserSecretName: acme-account
  challengeProvider:
    dns:
      provider: route53
      credentialSecretName: voyager-route53
  storage:
    secret:
      name: cert-kitecipro
```

For detailed guides on how to issue SSL certificates using Voyager, please see below:

- [Issue Let's Encrypt certificate using AWS Route53](/docs/guides/certificate/dns/route53.md)
- [Issue Let's Encrypt certificate using Google Cloud DNS](/docs/guides/certificate/dns/google-cloud.md)
