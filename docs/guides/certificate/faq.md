---
title: Certificate FAQ | Voyager
menu:
  product_voyager_v11.0.0:
    identifier: faq-certificate
    name: FAQ
    parent: certificate-guides
    weight: 25
product_name: voyager
menu_name: product_voyager_v11.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# FAQ

## Let's Encrypt FAQs

### How do I renew my LE certificates?
LE issues certificates that are valid for 90 days. Since 10.0.0 release, Voyager operator will try renewing your certificate 15 days (was 7 days in prior releases) prior to expiration. You can also [configure](https://github.com/appscode/voyager/pull/1316/files) how soon Voyager should try to renew the certificate by setting the `spec.renewalBufferDays` field in `Certificate` crd. Once renewed certificates are issued, HAProxy will be automatically updated to use the new certificates.

### I think I did everything according to this doc but my certificate is not issuing? How do I debug?
To debug, describe the certificate object and check the events listed under it. Voyager will report any warning events under the certificate object.

```console
kubectl describe certificate <name> --namespace <namespace>
```

You can also check the logs for voyager operator pod and look for anything suspicious.

```console
kubectl logs -f <voyager-pod-name> -n kube-system
```

### What are the rate limits for Let's Encrypt?
Please consult the official document on this matter: https://letsencrypt.org/docs/rate-limits/

### How to use Let's Encrypt staging servers?
If you are just testing Voyager and want to avoid hitting the rate limits in LE productoion environment, you have 2 options:

- Buy a cheap domain for testing. There are lot of $0.99/yr domains available these days.
- You can tell voyager to use the LE staging servers for issuing the certificate. The issued certificate is not trusted, hence should not be used in production websites. But this works great for testing purposes. To use the staging environment, set the key `ACME_SERVER_URL` in your acme secret in addition to your email address.

```console
kubectl create secret generic acme-account \
  --from-literal=ACME_EMAIL=me@example.com \
  --from-literal=ACME_SERVER_URL=https://acme-staging-v02.api.letsencrypt.org/directory
```

### Where is my LE account info?
Given your acme email and acme server url (if provided), voyager operator will open a new LE account. Voyager will store the account data in the acme user secret under `ACME_USER_PRIVATE_KEY` and `ACME_REGISTRATION_DATA` keys after the first successful registration. Any following interaction will LE will done using this account. This helps voyager to avoid performing repeated domain ownership challenged. We recommend that you keep a backup copy of the full secret. To be clear, if these keys are missing voyager will automatically register a new account with LE and use that.

```console
$ kubectl get secrets acme-account -o yaml
apiVersion: v1
data:
  ACME_EMAIL: dGFt29t
  ACME_REGISTRATION_DATA: eyJib2R5Ijp7InJlc291cmNlIjoicmVnIiwiaWQiOjI0OTc1NTYwLCJrZXkiOnsia3R5IjoiUlNBIiwibiI6IjNXRDRzY0hsUUN6N1JmbUZUNmZ3YXpIZ2UyNjhsajk5UGJmMkNwV1lSRzhlTFNHVGVBd0ZXdFVmRTRyMnItQkdjT3AtTnFtYUxBWGxGQmZTWjhtNzRnNEhPbHdPR0tYaTg1cG5hRkYxZS12MDEuYXBpLmxldHNlbmNyeXB0Lm9yZy9hY21lL25ldy1hdXRoeiIsInRlcm1zX29mX3NlcnZpY2UiOiJodHRwczovL2xldHNlbmNyeXB0Lm9yZy9kb2N1bWVudHMvTEUtU0EtdjEuMi1Ob3ZlbWJlci0xNS0yMDE3LnBkZiJ9
  ACME_USER_PRIVATE_KEY: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBM1dENHNjSGxRQ3o3UmZtRlQ2ZndhekhnZTI2OGxqOTlQYmYyQ3BXWVJHOGVMU0dUCmVBd0ZXdFVmRTRyMnIrQkdjT3ArTnFtYUxBWGxGQmZTWjhtNzRnNEhPbHdPR0tYaTg1cG5hRkYxU3hBL3BzNkMKMlZVK0tWQmtEczd6d200VmpZV1pXQUl1cDJPT3QxQjhzSE1zbmpuYm82d1dUeVh0TWZINVBoSUFxYnl0dUVKVgpWSklzUVh3WittaWVzOG9URUdIVjRldUgwVC9aL1NSZXpRNExUVExxN0UxNGZtK3FyOFV4b2FxTVhtSHFhNFA0b2svWWg0RHdieTFpelU1cDg9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
kind: Secret
metadata:
  creationTimestamp: 2017-11-27T23:44:32Z
  name: acme-account
  namespace: default
  resourceVersion: "33187"
  selfLink: /api/v1/namespaces/default/secrets/acme-account
  uid: eab30248-d3cc-11e7-8b04-02cf95c35e16
type: Opaque
```


### How can I distribute the issued ssl certificates?
There are several options:

- If you are trying to distribute the same ssl certificate across different namespaces of a cluster, you can use a tool like [kubed](https://appscode.com/products/kubed).
- If you want to distribute the issued certificates across different clusters, you can setup Voyager to issue certificates independently on each cluster. Please read the rate limiting restrictions for LE. The other option is to use [kubed](https://appscode.com/products/kubed).
- Just manually copy paste the `tls-***` secret to your destination cluster or namespace.


### How to issue certificate with multiple domains?
The above example shows how to issue a SANS certificate with multiple domains. The only restriction is that all domains must be using the same DNS provider account. They can use different domain registrars.

### How to issue Wildcard certificates via Let's Encrypt?
Voyager supports issuing wildcard certificates using Let's Encrypt since version 7.0.0. To issue wildcard domain, set the domain name in your certificate crd as `"*.yourdomain.com"`. Please note that wildcard domain is only supported with DNS challenges and can't be issued via HTTP challenge.

### Does Voyager support OCSP stapling?
Voyager currently does not issue certificates that use OCSP stapling. See [here](https://github.com/appscode/voyager/issues/531) for prior discussions.
