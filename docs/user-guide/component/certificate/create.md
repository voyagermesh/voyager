## Creating a Certificate
Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object. This
tutorial will walk you through creating certificate objects based on the googledns.

### Create Service Account Secret
Voyager requires Service account secret for your specified dns provider. This Secret spec is briefly described [here](provider.md).

### Create a Kubernetes Certificate Object
```yaml
apiVersion: appscode.com/v1beta1
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
```

In this example the domains DNS providers are `googlecloude`. Example Test `test-gcp-secret` should look like
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

```sh
kubectl create -f example.yaml
```

After submitting the Certificate configuration to the Kubernetes API it will be processed by the Voyager. You can view the process logs via
```
kubectl logs -f appscode-voyager
```

### Results
This object will create a certificate named `cert-test-cert`.

```sh
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

### Create Certificate with HTTP Provider

Your ingress must be present before a certificate create request. Your ingress must contains the required [TLS](/docs/user-guide/ingress/tls.md) fields.
```
  tls:
    - secretName: cert-test-cert
      hosts:
      - foo.example.com
  rules:
  - host: foo.example.com
```

As an example
```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: base-ingress
  namespace: foo
spec:
  tls:
    - secretName: cert-test-cert
      hosts:
      - foo.example.com
  rules:
  - host: foo.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: 80
```
When Your Ingress in ready. Set ingress IP/CNAME as your domain's A record. And Create a
certificate resource.

To Create a certificate with HTTP Provider you need to provide a ingress reference to certificate.
```yaml
apiVersion: appscode.com/v1beta1
kind: Certificate
metadata:
  name: test-cert
  namespace: default
spec:
  domains:
  - foo.example.com
  - bar.example.com
  email: jon.doe@example.com
  provider: http
  httpProviderIngressReference:
    apiVersion: appscode.com/v1beta1
    kind: Ingress
    Namespace: foo
    Name: base-ingress
```

When Your Certificate is ready you can find it out following these steps.

```sh
kubectl get secrets cert-test-cert
```

```
NAME      TYPE                DATA      AGE
cert-test-cert    kubernetes.io/tls   2         20m
```
