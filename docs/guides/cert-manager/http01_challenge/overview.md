---
title: Issue Let's Encrypt certificate using HTTP-01 challenge with cert-manager
description: Issue Let's Encrypt certificate using HTTP-01 challenge with cert-manager in Kubernetes
menu:
  product_voyager_10.0.0:
    identifier: overview-http-cert-manager
    name: Overview
    parent: http01-cert-manager
    weight: 10
product_name: voyager
menu_name: product_voyager_10.0.0
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Issue Let's Encrypt certificate using HTTP-01 challenge with cert-manager

## 1. Setup Issuer/ClusterIssuer

Setup a [ClusterIssuer (Or Issuer)](/docs/guides/cert-manager/get-started.md) for your Ingress by applying
this [clusterissuer.yaml](/docs/examples/cert-manager/http/clusterissuer.yaml)

<!-- https://docs.cert-manager.io/en/latest/tasks/issuers/setup-acme/http01/index.html -->

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # You must replace this email address with your own.
    # Let's Encrypt will use this to contact you about expiring
    # certificates, and issues related to your account.
    email: user@example.com
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      # Secret resource used to store the account's private key.
      name: example-issuer-account-key
    # Add a single challenge solver, HTTP01 using nginx
    solvers:
      - http01:
          ingress:
            name: test-ingress
```

Here `test-ingress` is the name of ingress you're going to create.

**IngressClass or IngressName?**

If the ingressClass field is specified, cert-manager will create new Ingress resources in order to route traffic to the ‘acmesolver’ pods, which are responsible for responding to ACME challenge validation requests. If the `ingress.name` field is specified, cert-manager will edit the named ingress resource in order to solve HTTP01 challenges. Since Voyager allocates a separate external IP for each Ingress resource, use `ingress.name` mechanism for Voyager.

## 2. Create Ingress

We are going to use a nginx server as the backend. To deploy nginx server, run the following commands:

```console
kubectl run nginx --image=nginx
kubectl expose deployment nginx --name=web --port=80 --target-port=80
```

Now create your ingress by applying [ingress.yaml](/docs/examples/cert-manager/http/ingress.yaml)

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    kubernetes.io/ingress.class: voyager
    certmanager.k8s.io/cluster-issuer: "letsencrypt-staging"
    certmanager.k8s.io/acme-challenge-type: http01
spec:
  tls:
    - hosts:
        - kiteci.appscode.ninja
      secretName: quickstart-kiteci-tls
  rules:
    - http:
        paths:
          - backend:
              serviceName: web
              servicePort: 80
            path: /
```

Then you'll see that a Certificate crd is created automatically for this ingress

```console
kubectl get certificates.certmanager.k8s.io --all-namespaces
```

But the certificate is still invalid.

Now take the `EXTERNAL-IP` from the corresponding service:

```console
kubectl get svc
```

```console
NAMESPACE       NAME                                          TYPE           CLUSTER-IP     EXTERNAL-IP       PORT(S)                      AGE
default         voyager-test-ingress                        LoadBalancer   10.7.249.7     35.239.22.162     80:31919/TCP,443:32751/TCP   44s
```

Create an A-record for `kiteci-dns.appscode.ninja` mapped to `35.239.22.162`.

Wait till this is resolved:

```console
dig +short kiteci-dns.appscode.ninja
```

Describe that certificate and wait until you see `Certificate issued successfully` when you describe the certificate.

```console
kubectl describe certificates.certmanager.k8s.io quickstart-kiteci-tls
```

Let’s Encrypt does not support issuing wildcard certificates with HTTP-01 challenges. To issue wildcard certificates, you must use the DNS-01 challenge.

The dnsNames field specifies a list of Subject Alternative Names to be associated with the certificate. If the commonName field is omitted, the first element in the list will be the common name.
