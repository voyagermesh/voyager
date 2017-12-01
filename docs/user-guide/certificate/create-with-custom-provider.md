---
title: Create Certificate with Custom Provider
menu:
  product_voyager_5.0.0-rc.5:
    identifier: create-with-custom-provider
    name: Custom Provider
    parent: certificate
    weight: 20
product_name: voyager
left_menu: product_voyager_5.0.0-rc.5
section_menu_id: user-guide
---

## Creating a Certificate
Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object. This
tutorial will walk you through creating certificate from custom ACME server.

### Create Service Account Secret
Voyager requires Service account secret for your specified dns provider. This Secret spec is briefly described [here](provider.md).

### Create a Kubernetes Certificate Object
The following example will create a certificate from Lets Encrypt Staging.

Create the DNS provider secret first
```yaml
kind: Secret
metadata:
  name: test-gcp-secret
  namespace: default
data:
  GCE_PROJECT: <project-name>
  GCE_SERVICE_ACCOUNT_DATA: <service-account-json>
```

Create ACME User Secret with key ACME_EMAIL.
```yaml
kind: Secret
metadata:
  name: test-user-secret
  namespace: default
data:
  ACME_EMAIL: test@appscode.com
  ACME_SERVER_URL: https://acme-staging.api.letsencrypt.org/directory
```

Create the Certificate resource.
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Certificate
metadata:
  name: test-cert
  namespace: default
spec:
  domains:
  - foo.example.com
  - bar.example.com
  acmeUserSecretName: test-user-secret
  challengeProvider:
    dns:
      provider: googlecloud
      credentialSecretName: test-gcp-secret
  storage:
    secret: {}
```
```console
kubectl create -f hack/example/certificate.yaml
```

After submitting the Certificate configuration to the Kubernetes API it will be processed by the Voyager. You can view the process logs via
```
kubectl logs -f appscode-voyager
```

### Results
This object will create a secret named `tls-test-cert`. This certificate will created from the custom acme server
that is provided in the secret.

```console
kubectl get secrets tls-test-cert
```

```
NAME      TYPE                DATA      AGE
tls-test-cert    kubernetes.io/tls   2         20m
```

```
kubectl describe secrets tls-test-cert
```

```
Name:           tls-test-cert
Namespace:      default

Type:   kubernetes.io/tls

Data
====
tls.crt:        3411 bytes
tls.key:        1679 bytes
```
