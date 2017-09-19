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

Create the Creatificate object
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
  email: jon.doe@example.com
  provider: googlecloud
  providerCredentialSecretName: test-gcp-secret
  acmeStagingURL: <Your custom ACME Server URL>
```

For testing purpose you may use Let's Encrypt's staging URL `https://acme-staging.api.letsencrypt.org/directory` as `acmeStagingURL`

In this example the DNS provider is `googlecloud`. To see the full list of supported providers, visit [here](provider.md) .

```console
kubectl create -f hack/example/certificate.yaml
```

After submitting the Certificate configuration to the Kubernetes API it will be processed by the Voyager. You can view the process logs via
```
kubectl logs -f appscode-voyager
```

### Results
This object will create a certificate named `cert-test-cert`.

```console
kubectl get secrets cert-test-cert
```

```
NAME      TYPE                DATA      AGE
cert-test-cert    kubernetes.io/tls   2         20m
```

```
kubectl describe secrets cert-test-cert
```

```
Name:           cert-test-cert
Namespace:      default

Type:   kubernetes.io/tls

Data
====
tls.crt:        3411 bytes
tls.key:        1679 bytes
```

> HTTP provider certificate can be also be created applying annotations to ingress. But you can't create a
certificate and mount the secret same time in the ingress.
