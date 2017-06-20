# User Guide
This guide will walk you through deploying the Voyager controller.

### Deploying Voyager
Voyager controller communicates with kube-apiserver at inCluster mode if no master or kubeconfig is provided. It watches Ingress and Certificate resource
to handle corresponding events.

```sh
$ export CLOUD_PROVIDER=<provider-name> // ie:
                                        // - gce
                                        // - gke
                                        // - aws
                                        // - azure


$ export CLOUDE_CONFIG=<path>           // The path to the cloud provider configuration file.
                                        // Empty string for no configuration file.
                                        // ie. for azure use /etc/kubernetes/azure.json


$ export TAG=3.0.0                      // Docker image tag for Voyager operator.

$ curl https://raw.githubusercontent.com/appscode/voyager/master/hack/deploy/deployments.yaml | \
        envsubst | \
        kubectl apply -f -
```

Once Controller is *Running* It will create the [required ThirdPartyResources for ingress and certificates](/docs/developer-guide#third-party-resources).
Check the Controller is running or not via `kubectl get pods` there should be a pod nameed `appscode-voyager-xxxxxxxxxx-xxxxx`.
Now Create Your Ingress/Certificated.


#### Configuration Options
```
      --address string                        Address to listen on for web interface and telemetry. (default ":56790")
      --analytics                             Send analytical event to Google Analytics (default true)
  -c, --cloud-provider string                 Name of cloud provider
      --cloud-config string                   The path to the cloud provider configuration file.  Empty string for no configuration file.
      --haproxy-image string                  haproxy image name to be run (default "appscode/haproxy:1.7.6-3.0.0")
      --haproxy.server-metric-fields string   Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1 (default "2,3,4,5,6,7,8,9,13,14,15,16,17,18,21,24,33,35,38,39,40,41,42,43,44")
      --haproxy.timeout duration              Timeout for trying to get stats from HAProxy. (default 5s)
  -h, --help                                  help for run
      --ingress-class string                  Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.
      --kubeconfig string                     Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                         The address of the Kubernetes API server (overrides any value in kubeconfig)
```

Voyager can run HAProxy in 3 different modes. `cloude-provider` flag should be set appropriately depending on the mode. These modes are:

- LoadBalancer: In this mode, a Kubernetes LoadBalancer type service is used to expose HAProxy to the internet.
This is supported for cloud providers known to Kubernetes (`aws`, `gce` and `azure`), `--cloud-provider` flag is required to properly setup this loadbalancer. This mode supports reserved ip on GCE.

- HostPort: In this mode, HAProxy is run as DaemonSet using nodeSelector and hostNetwork:true. As a result,
HAProxy's IP will be same as the IP address for nodes where it is running. This is supported on any cloud provider
(known or unknown to Kubernetes). Voyager will open firewall, if a `cloud-provider` is one of `aws`, `gce`, `gke` or
`azure`. If cloud provider is unknown (say, running on DigitalOcean), users are required to configure firewall as needed.

- NodePort: In this mode, a Kubernetes NodePort type service is used to expose HAProxy to the internet. This is supported on any cloud provider including
baremetal clusters. Users are required to configure firewall as needed. --cloud-provider flag can be left unset, if used with providers other than `aws`, `gce`, `gke` or
`azure`.

You can choose the mode in your Ingress YAML using label: [ingress.appscode.com/type](/docs/user-guide/ingress#configurations-options)

## Run with helm
You can deploy Voyager operator with helm by using this [chart](/chart/voyager/README.md).

## Ingress
This resource Type is backed by an controller which monitors and manages the resources of AppsCode Ingress Kind. Which is used for maintain and HAProxy backed loadbalancer to the cluster for open communications inside cluster from internet via the loadbalancer.
Even when a resource for AppsCode Ingress type is created, the controller will treat it as a new load balancer request and will create a new load balancer, based on the configurations.

### Resource
A minimal AppsCode Ingress resource Looks like at the kubernetes level:

```yaml
apiVersion: networking.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - path: '/testPath'
        backend:
          serviceName: test-service
          servicePort: '80'
          headerRule:
          - X-Forwarded-Host %[base]
          rewriteRule:
          - '^([^\\ :]*)\\ /(.*)$ \\1\\ /testings/\\2'
          backendRule:
          - 'acl add_url capture.req.uri -m beg /test-second'
          - 'http-response set-header X-Added-From-Proxy added-from-proxy if add_url'
```

POSTing this to Kubernetes, API server will need to create a loadbalancer.

**Line 1-3**: With all other Kubernetes config, AppsCode Ingress resource needs `apiVersion`, `kind` and `metadata` fields.
`apiVersion` and `kind` needs to be exactly same as `networking.appscode.com/v1beta1`, and, `specific version` currently as `v1beta1`, to identify the resource
as AppsCode Ingress. In metadata the `name` and `namespace` indicates the resource identifying name and its Kubernetes namespace.


**Line 6**: Ingress spec has all the information needed to configure a loadbalancer. Most importantly, it contains
a list of rules matched against all incoming requests.

**Line 9**: Each `http rule` contains the following information: A host (eg: foo.bar.com, defaults to *), a
list of paths (eg: /testPath), each of which has a backend associated (test:80). Both the host and path must
match content of an incoming request before the loadbalancer directs traffic to backend.

**Line 12-13**: A backend is a service:port combination as described in the services doc. Ingress traffic is
typically sent directly to the endpoints matching a backend.

**Line 14-15**: `headerRule` are a list of rules applied to the `request header` before sending it to desired backend. For simplicity the header rules are formatted with respect to HAProxy.

**Line 16-17**: `rewriteRule` are a list of rules to be applied in the request URL. It can append, truncate or rewrite
the request URL. These rules also follow `HAProxy` rewrite rule formats.

**Line 18-20**: `backendRule` are a list of rules to be applied in the backend. It supports full
spectrum of HAProxy rules.

**Other Parameters**: For the sake of simplicity, the example Ingress has no global config parameters,
tcp load balancer and tls terminations. We will discuss those later. One can specify a global **default backend**
in absence of those requests which doesn’t match a rule in spec, are sent to the default backend.

### The Endpoints are like:

|  VERB   |                     ENDPOINT                                | ACTION | BODY
|---------|-------------------------------------------------------------|--------|-------
|  GET    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates          | LIST   | nil
|  GET    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | GET    | nil
|  POST   | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates          | CREATE | JSON
|  PUT    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | UPDATE | JSON
|  DELETE | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | DELETE | nil

### Dive into Ingress
You can learn more about ingress options by reading [this doc](ingress/README.md).


## Certificate
Certificate objects are used to declare one or more Let's Encrypt issued TLS certificates. Cetificate objects are consumed by the Voyager controller.
Before you can create a Certificate object you must create the Certificate Third Party Resource in your Kubernetes cluster.

### Resource
A minimal Certificate resource looks like at the kubernetes level:

```yaml
apiVersion: networking.appscode.com/v1beta1
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

POSTing this to Kubernetes, API server will create a certificate and store it as a secret that can be used to SSL with ingress.

**Line 1-3**: With all other Kubernetes config, AppsCode Ingress resource needs `apiVersion`, `kind` and `metadata` fields.
`apiVersion` and `kind` needs to be exactly same as `networking.appscode.com/v1beta1`, and, `specific version` currently as `v1beta1`, to identify the resource
as AppsCode Ingress. In metadata the `name` and `namespace` indicates the resource identifying name and its Kubernetes namespace.

**Line 7-9**: domains specifies the domain list that the certificate needs to be issued. First on the list will be used as the
certificate common name.

**Line 10**: The email address used for a user registration.

**Line 11**: The name of the dns provider.

**Line 12**: DNS provider credential that will be used to configure the domains.

### The Endpoints are like:

|  VERB   |                     ENDPOINT                                    | ACTION | BODY
|---------|-----------------------------------------------------------------|--------|-------
|  GET    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates          | LIST   | nil
|  GET    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | GET    | nil
|  POST   | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates          | CREATE | JSON
|  PUT    | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | UPDATE | JSON
|  DELETE | /apis/networking.appscode.com/v1beta1/namespace/`ns`/certificates/`name`   | DELETE | nil

### Dive into Certificates
You Can Learn more about issuing SSL certificates by reading [this doc](certificate/README.md).

## Running Voyager alongside with other ingress controller
Voyager can be configured to handle default kubernetes ingress or only ingress.appscode.com. Voyager can also be run
along side with other controllers.
Read the example how to use [HTTP Provider](/docs/user-guide/certificate/create.md#create-certificate-with-http-provider)
for certificate.

```sh
  --ingress-class
  // this flag can be set to 'voyager' to handle only ingress
  // with annotation kubernetes.io/ingress.class=voyager.

  // If unset, Voyager will also handle ingress without ingress-class annotation.
```
Other ingress controller can be run alongside Voyager to handle specific classed ingress.
