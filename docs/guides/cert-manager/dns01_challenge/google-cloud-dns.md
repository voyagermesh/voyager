---
title: Issue Let's Encrypt certificate using Google Cloud DNS
description: Issue Let's Encrypt certificate using Google Cloud DNS in Kubernetes
menu:
  product_voyager_10.0.0:
    identifier: google-cloud-dns-cert-manager
    name: Google Cloud DNS
    parent: dns01-cert-manager
    weight: 15
product_name: voyager
menu_name: product_voyager_10.0.0
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Issue Let's Encrypt certificate using Google Cloud DNS

This tutorial shows how to issue free SSL certificate from Let's Encrypt via DNS challenge for domains using Google Cloud DNS service.

This article has been tested with a GKE cluster.

```console
$ kubectl version --short
Client Version: v1.8.8
Server Version: v1.8.8-gke.0
```

## 1. Setup Issuer/ClusterIssuer

Now create a service account from your Google Cloud Console

![svcac1](/docs/images/cert-manager/google_dns/svcac1.png)
![svcac2](/docs/images/cert-manager/google_dns/svcac2.png)
![svcac3](/docs/images/cert-manager/google_dns/svcac3.png)

Then create a Kubernetes Secret with this Service Account:

```console
kubectl create secret generic clouddns-service-account --from-file=service-account.json=<path-to-json-file>
```

Now create this issuer by applying [issuer.yaml](/docs/examples/cert-manager/google_cloud/issuer.yaml)

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: letsencrypt-staging-dns
  namespace: default
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: example@kite.com
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: example-issuer-account-key
    solvers:
      - dns01:
          clouddns:
            # A secretKeyRef to a google cloud json service account
            serviceAccountSecretRef:
              name: clouddns-service-account
              key: service-account.json
            # The project in which to update the DNS zone
            project: test-cert
```

## 2. Create Ingress

We are going to use a nginx server as the backend. To deploy nginx server, run the following commands:

```console
kubectl run nginx --image=nginx
kubectl expose deployment nginx --name=web --port=80 --target-port=80
```

Now, Create [ingress.yaml](/docs/examples/cert-manager/google_cloud/ingress.yaml)

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress-deploy-k8s-dns
  namespace: default
  annotations:
    kubernetes.io/ingress.class: voyager
    certmanager.k8s.io/issuer: "letsencrypt-staging-dns"
    certmanager.k8s.io/acme-challenge-type: dns01
spec:
  tls:
    - hosts:
        - kiteci-dns.appscode.ninja
      secretName: kiteci-dns-tls
  rules:
    - host: kiteci-dns.appscode.ninja
      http:
        paths:
          - backend:
              serviceName: web
              servicePort: 80
            path: /
```

Then take the `EXTERNAL-IP` from the corresponding service:

```console
kubectl get svc
```

```console
NAME                                          TYPE           CLUSTER-IP     EXTERNAL-IP       PORT(S)                      AGE
voyager-test-ingress-deploy-k8s-route53-dns   LoadBalancer   10.7.248.189   35.225.111.106    443:30713/TCP,80:31137/TCP   21m
```

Create an A-record for `kiteci-dns.appscode.ninja` mapped to `35.225.111.106` with Google DNS.

Wait until you can see it resolved:

```console
dig +short kiteci-dns.appscode.ninja
```

## 3. Create Certificate

Then create this [certificate.yaml](/docs/examples/cert-manager/google_cloud/certificate.yaml)

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  name: kiteci-dns
  namespace: default
spec:
  secretName: kiteci-dns-tls
  issuerRef:
    name: letsencrypt-staging-dns
  dnsNames:
    - kiteci-dns.appscode.ninja
```

Now, List the certificates and describe that certificate and wait until you see `Certificate issued successfully` when you describe the certificate.

```console
kubectl get certificates.certmanager.k8s.io --all-namespaces
```

Then visit `kiteci-dns.appscode.ninja` from browser and check the certificate that it was issued from let's encrypt. (For let's encrypt staging environment, you will see that the certificate was issued by `Fake LE Intermediate X1`.)
