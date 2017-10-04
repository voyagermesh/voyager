## Creating a Certificate
Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object. This
tutorial will walk you through creating certificate objects based on the googledns.

### Create Service Account Secret
Voyager requires Service account secret for your specified dns provider. This Secret spec is briefly described [here](provider.md).

### Create a Kubernetes Certificate Object
The following example will create a certificate from Lets Encrypt Prod. If you want to create a certificate from
another ACME server [see this example](create-with-custom-provider.md)

Create the DNS provider secret first.
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
    secretStore: {}
```
In this example the domains DNS providers are `googlecloud`.

See the Supported Providers List [here](provider.md)

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

### Create Certificate with HTTP Provider

Your ingress must be **running before a certificate can be issues** from Let's Encrypt using HTTP validation.
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: base-ingress
  namespace: foo
spec:
  rules:
  - host: foo.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: 80
```
Now create the ingress. Once the ingress is configured, it will show a IP address or CNAME in the ingres status.
Now, go to your domain registrar's website and set IP/CNAME for your domain(s). Now, you are ready to issue a SSL certificate using
HTTP provider.

Below is an example certificate definition. Please note that to use HTTP Provider, you need to point to the ingress created in above.
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
    http:
      ingress:
        apiVersion: voyager.appscode.com/v1beta1
        kind: Ingress
        name: base-ingress
        namespace: default
  storage:
    secretStore: {}
```


When your certificate is issued, you will see a `kubernetes.io/tls` type secret.

```console
kubectl get secrets cert-test-cert
```

```
NAME      TYPE                DATA      AGE
cert-test-cert    kubernetes.io/tls   2         20m
```
