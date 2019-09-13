---
title: TLS | Kubernetes Ingress
menu:
  product_voyager_{{ .version }}:
    identifier: overview-tls
    name: Overview
    parent: tls-ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# TLS
You can secure an Ingress by specifying a secret containing TLS pem or by referring a `certificates.voyager.appscode.com` resource.
`certificates.voyager.appscode.com` can manage an certificate resource and use that certificate to encrypt communication.

This tutorial will show you how to secure an Ingress using TLS/SSL certificates.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

Now, install Voyager operator in your `minikube` cluster following the steps [here](/docs/setup/install.md).

```console
minikube start
./hack/deploy/minikube.sh
```

To keep things isolated, this tutorial uses a separate namespace called `demo` throughout this tutorial. Run the following command to prepare your cluster for this tutorial:

```console
$ kubectl create namespace demo
namespace "demo" created
```

## Sourcing TLS Certificate

You can use an existing TLS certificate/key pair or use Voyager to issue free SSL certificates from Let's Encrypt.

### Import Existing Certificate

To import an existing TLS certificate/key pair into a Kubernetes cluster, run the following command.

```console
$ kubectl create secret tls tls-secret --namespace=demo --cert=path/to/tls.cert --key=path/to/tls.key
secret "tls-secret" created
```

This will create a Secret with the PEM formatted certificate under `tls.crt` key and the PEM formatted private key under `tls.key` key.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tls-secret
  namespace: demo
data:
  tls.crt: base64 encoded cert
  tls.key: base64 encoded key
```

### Issue Certificates from Let's Encrypt

To issue a free TLS/SSL certificate from Let's Encrypt, create a `Certificate` object with the list of domains. To learn more, please visit the following links:

- [Using HTTP-01 challenge](/docs/guides/certificate/http/overview.md)
- [Using DNS-01 challenge](/docs/guides/certificate/dns/providers.md)

## Secure HTTP Service

To terminate a HTTP service,

Caveats:
- You can't terminate default backend

For HTTP, If the `spec.TLS` section in an Ingress specifies different hosts, they will be multiplexed
on the same port according to hostname specified through SNI TLS extension (Voyager supports SNI).

For handling wildcard domains use **"\*"** as hostname ( [Example](https://github.com/tamalsaha/voyager-wildcard/blob/master/mrasero/ing-https.yaml) )

Referencing this secret in an Ingress will tell the Voyager to secure the channel from client to the loadbalancer using TLS:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: demo
spec:
  tls:
  - secretName: tls-secret
    hosts:
    - one.example.com
  rules:
  - host: one.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
```
This Ingress will open an `https` listener to secure the channel from the client to the loadbalancer,
terminate TLS at load balancer with the secret retried via SNI and forward unencrypted traffic to the
`test-service`.

## Secure TCP Service

Adding a TCP TLS termination at Voyager Ingress is slightly different than HTTP, as TCP mode does not have
SNI support. A TCP endpoint with TLS termination, will look like this in Voyager Ingress:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: demo
spec:
  tls:
    - secretName: testsecret
      hosts:
      - appscode.example.com
  rules:
  - host: appscode.example.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```
You need to set  the secretName field with the TCP rule to use a certificate.

## FAQ

**Q: How to serve both TLS and non-TLS under same host?**

Voyager Ingress can support for TLS and non-TLS traffic for same host in both HTTP and TCP mode. To do that you need to specify `noTLS: true` for the corresponding rule. Here is an example:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: demo
spec:
  tls:
  - secretName: onecert
    hosts:
    - one.example.com
  rules:
  - host: one.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  - host: one.example.com
    http:
      noTLS: true
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  - host: one.example.com
    tcp:
      port: '7878'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
  - host: one.example.com
    tcp:
      port: '7800'
      noTLS: true
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```

For this Ingress, HAProxy will open up 3 separate ports:

- port 443: This is used by `spec.rules[0]`. Passes traffic to pods behind test-server:80. Uses TLS, since `spec.TLS` has a matching host.

- port 80: This is used by `spec.rules[1]`. Passes traffic to pods behind test-server:80. __Uses no TLS__, even though `spec.TLS` has a matching host. This is because `http.noTLS` is set to true for this rule.

- port 7878: This is used by `spec.rules[2]`. Passes traffic to pods behind tcp-service:50077. Uses TLS, since `spec.TLS` has a matching host.

- port 7880: This is used by `spec.rules[3]`. Passes traffic to pods behind tcp-service:50077. __Uses no TLS__, even though `spec.TLS` has a matching host. This is because `tcp.noTLS` is set to true for this rule.

## Cleaning up

To cleanup the Kubernetes resources created by this tutorial, run:

```console
kubectl delete ns demo
```

If you would like to uninstall Voyager operator, please follow the steps [here](/docs/setup/uninstall.md).
