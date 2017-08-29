## Creating a Certificate
Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object. This
tutorial will walk you through creating certificate from custom ACME server.

### Create Service Account Secret
Voyager requires Service account secret for your specified dns provider. This Secret spec is briefly described [here](provider.md).

### Create a Kubernetes Certificate Object
The following example will create a certificate from Lets Encrypt Staging.

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

In this example the domains DNS providers are `googlecloud`. Example Test `test-gcp-secret` should look like
```yaml
kind: Secret
metadata:
  name: ssl-appscode-io
  namespace: default
data:
  GCE_PROJECT: <project-name>
  GOOGLE_APPLICATION_CREDENTIALS: <credential>
```

See the Supported Providers List [here](provider.md)

```console
kubectl create -f example.yaml
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
