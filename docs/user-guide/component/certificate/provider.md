# Configuring your challenge provider(s)

## DNS Providers
Voyager uses kubernetes secret within the pod to fetch credentials required for various DNS providers.
Making those correctly accessible to Voyager will require specifing the secret name inside an certificate objects.
The Secret will need the Key name exactly provided.

### HTTP
HTTP Provider will requires an Ingress refrence to resolve with. Reference an Ingress name for http provider.

### Cloudflare
`CLOUDFLARE_EMAIL`: The email of the cloudflare user
`CLOUDFLARE_API_KEY`: The API key corresponding to the email

### Digital Ocean
`DO_AUTH_TOKEN`: The digital ocean authorization token

### DNSimple
`DNSIMPLE_EMAIL`: The email fo the DNSimple user
`DNSIMPLE_API_KEY`: The API key corresponding to the email

### DNS Made Easy
`DNSMADEEASY_API_KEY`: The API key for DNS Made Easy
`DNSMADEEASY_API_SECRET`: The api secret corresponding with the API key
`DNSMADEEASY_SANDBOX`: A boolean flag, if set to true or 1, requests will be sent to the sandbox API

### Dyn
`DYN_CUSTOMER_NAME`: The customer name of the Dyn user
`DYN_USER_NAME`: The user name of the Dyn user
`DYN_PASSWORD`: The password of the Dyn user

### Gandi
`GANDI_API_KEY`: The API key for Gandi

### Google Cloud
`GCE_PROJECT`: The name of the Google Cloud project to use
`GOOGLE_APPLICATION_CREDENTIALS`: Credential Data.

### Namecheap
`NAMECHEAP_API_USER`: The username of the namecheap user
`NAMECHEAP_API_KEY`: The API key corresponding with the namecheap user

### OVH
`OVH_ENDPOINT`: The URL of the API endpoint to use
`OVH_APPLICATION_KEY`: The application key
`OVH_APPLICATION_SECRET`: The secret corresponding to the application key
`OVH_CONSUMER_KEY`: The consumer key

### PDNS
`PDNS_API_KEY`: The API key to use

### RFC2136
The rfc2136 provider works with any DNS provider implementing the DNS Update rfc2136.
the TSIG variables need only be set if using TSIG authentication.

`RFC2136_NAMESERVER`: The network address of the provider, in the form of "host" or "host:port"
`RFC2136_TSIG_ALGORITHM`: The algorithm to use for TSIG authentication.
`RFC2136_TSIG_KEY`: The key to use for TSIG authentication.
`RFC2136_TSIG_SECRET`: The secret to use for TSIG authentication.

### Amazon Route53
`AWS_ACCESS_KEY_ID`: The access key ID
`AWS_SECRET_ACCESS_KEY`: The secret corresponding to the access key

### Vultr
`VULTR_API_KEY`: The API key to use

### Linode
`LINODE_API_KEY`: API Key for linode to use.