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

Your Ingress must present before an certificate create request via that Ingress.

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

You need to set your domain's A Record to that Ingress IP / CNAME. That way you can create certificate with http providers.