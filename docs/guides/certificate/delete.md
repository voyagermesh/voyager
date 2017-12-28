---
menu:
  product_voyager_5.0.0-rc.9:
    identifier: certificate-delete
    name: Delete
    parent: certificate
    weight: 60
product_name: voyager
menu_name: product_voyager_5.0.0-rc.9
section_menu_id: guides
---
# Deleting Certificate

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
