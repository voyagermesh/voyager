---
title: Delete Certificate | Voyager
menu:
  product_voyager_8.0.0:
    identifier: delete-certificate
    name: Delete
    parent: certificate-guides
    weight: 20
product_name: voyager
menu_name: product_voyager_8.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Deleting Certificate

## Pausing Certificte

Voyager operator periodically (default 5 mins) checks each certificate whether whether it needs to be reissued. If you have a bad configuration (example, bad dns credentials), this can led to rate limit issued with Let's Encrypt. You can delete the certificate object to stop retries. Alternatively, you can mark the certificate object as `spec.paused: true`. This will cause Voyager operator to skip checking this certificate for renewals.

## Purging Certificate

Deleting a Kubernetes `Certificate` object will only delete the certificate CRD from Kubernetes.
It will not delete the obtained certificate and user account secret from Kubernetes. User have to manually delete these secrets for complete cleanup.

 - Delete Certificate crd.

```console
kubectl delete certificate.voyager.appscode.com test-cert
```

 - Delete Obtained Let's Encrypt tls certificate

```console
kubectl delete secret tls-test-cert
```

 - Delete Let's Encrypt user account `Secret`

```console
kubectl delete secret test-user-secret
```
