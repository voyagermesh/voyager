---
title: Get Started | Voyager
menu:
  product_voyager_10.0.0:
    identifier: get-started-cert-manager
    name: Get Started
    parent: cert-manager-guides
    weight: 10
product_name: voyager
menu_name: product_voyager_10.0.0
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

## 1. Install Cert-Manager

https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html

```console
kubectl create namespace cert-manager
kubectl label namespace cert-manager certmanager.k8s.io/disable-validation=true
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.8.1/cert-manager.yaml
```

Add `--validate=false` to the last command if your kubectl version is <= v1.12, like this:

```console
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.8.1/cert-manager.yaml --validate=false
```

## 2. Setup Issuer/ClusterIssuer

### [Supported Issuer](https://docs.cert-manager.io/en/latest/tasks/issuers/index.html)

These are the supported Certificate Issuers:

1. [acme](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-acme/index.html)
2. [ca](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-ca.html)
3. [self signed](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-selfsigned.html)
4. [vault](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-vault.html)
5. [venafi](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-venafi.html)

Here we will show issuing certificates from Let's Encrypt using ACME protocol. For others, click on the link for the respective issuers.

#### [acme](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-acme/index.html)

The ACME Issuer type represents a single Account registered with the ACME server. When you create a new ACME Issuer, cert-manager will generate a private key which is used to identify you with the ACME server. To set up a basic ACME issuer, you should create a new Issuer or ClusterIssuer resource.

### [Issuer](https://docs.cert-manager.io/en/latest/reference/issuers.html)

Issuers (and ClusterIssuers) represent a certificate authority from which signed x509 certificates can be obtained, 
such as Let’s Encrypt. You will need at least one Issuer or ClusterIssuer in order to begin issuing certificates within your cluster.

Like this [issuer.yaml](/docs/examples/cert-manager/issuer.yaml)

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: letsencrypt-prod
  namespace: edge-services
spec:
  acme:
    # The ACME server URL
    server: https://acme-v02.api.letsencrypt.org/directory
    # Email address used for ACME registration
    email: user@example.com
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-prod
    # Enable HTTP01 validations
    http01: {}
```

The `spec.email` will be used to register for your let's encrypt account and `privateKeySecretRef` will contain the private key of this account.

#### [ClusterIssuer](https://docs.cert-manager.io/en/latest/reference/clusterissuers.html)

An Issuer is a namespaced resource, and it is not possible to issue certificates from an Issuer in a different 
namespace. If you want to create a single issuer than can be consumed in multiple namespaces, you should consider 
creating a ClusterIssuer resource.

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
```

When referencing a Secret resource in ClusterIssuer resources (eg `spec.acme.solvers.dns01.cloudflare.apiKeySecretRef`) the Secret needs to be in the same namespace as the cert-manager controller pod. You can optionally override this by using the `--cluster-resource-namespace` argument to the controller.

### Let's Encrypt Production vs Staging Environment

For production use, use the Let's Encrypt Production API like above. For testing things out, you can use the Staging API as there is a rate limit for issuing certificates. Just replace the `spec.acme.server` like this

```yaml
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
```

In this doc, we used the staging api and as a result, you will see that the certificate was issued by `Fake LE Intermediate X1`.

For more to know, visit [here](https://letsencrypt.org/docs/rate-limits/)

### [Certificate Duration and Renewal Window](https://docs.cert-manager.io/en/latest/reference/certificates.html)

The default duration for all certificates is 90 days and the default renewal windows is 30 days. This means that certificates are considered valid for 3 months and renewal will be attempted within 1 month of expiration.

You can change that value using `duration` and `renewBefore` field in [certificate.yaml](/docs/examples/cert-manager/certificate.yaml),

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  name: example
spec:
  secretName: example-tls
  duration: 24h
  renewBefore: 12h
  dnsNames:
    - foo.example.com
    - bar.example.com
  issuerRef:
    name: my-internal-ca
    kind: Issuer
```

That means, this certificate's validity period is 24 hours and it will begin trying to renew 12 hours before the certificate expiration.
