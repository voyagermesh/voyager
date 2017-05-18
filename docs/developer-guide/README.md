## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Voyager.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Voyager should not change.

### Building Voyager on local environment
Some of the Voyager development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

### Go development environment
Voyager is written in the go programming language. The release is built and tested on **go 1.7.5**. If you haven't set up a Go
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install the go tools.

```sh
go get github.com/appscode/voyager
cd $GOPATH/src/github.com/appscode/voyager
```

### Dependency management
Voyager build and test scripts use glide to manage dependencies.

To install glide follow [these instructions](https://github.com/Masterminds/glide#install).

Currently the project includes all its required dependencies inside `vendor` to make things easier.

```sh
glide slow
```

### Run Test
#### Run Short Unit Test by running
```sh
go test -v ./cmd/... ./pkg/...
```

#### Run Full Test
To Run Full unit test You need to provide some secret in `hack/configs/.env` file. Or You may add them as
environment variables.
```sh
TEST_GCE_SERVICE_ACCOUNT_DATA
TEST_GCE_PROJECT
TEST_ACME_USER_EMAIL
TEST_DNS_DOMAINS
```
Then run
```sh
$ ./hack/make.py test unit
```

#### Run e2e Test
```
$ ./hack/make.py test minikube // Run Test against minikube, this requires minikube to be set up and started.

$ ./hack/make.py test e2e -cloud-provider=gce -cluster-name=autobots // Test e2e against gce cluster

$ ./hack/make.py test integration -cloud-provider=gce -cluster-name=autobots // Run Integration test against gce
                                                                             // This requires voyager to be deployed in the cluster.

```

```
- Run only one e2e test
$ ./hack/make.py test e2e -cloud-provider=gce -cluster-name=autobot -test-only=CoreIngress


- Run One test but do not delete all resource that are created
$ ./hack/make.py test minikube -cloud-provider=gce -cluster-name=autobot -test-only=CoreIngress -cleanup=false


- Run Service IP Persist test with provided IP
$ ./hack/make.py test e2e -cloud-provider=gce -cluster-name=autobot -test-only=CreateIPPersist -lb-ip=35.184.104.215

```

### Local Build
To build Voyager using your local Go development environment (generate linux binaries):
```sh
$ ./hack/make.py build
```
Read full [Build instructions](build.md).

<br><br>
## Architecture
Voyager works by implementing third party resource data watcher for kubernetes. It connects with k8s apiserver
for specific events as ADD, UPDATE and DELETE. and perform required operations.

Ingress watcher generates the configuration for HAProxy and stores it as a ConfigMaps and creates a RC with
specified HAProxy - that is configured with auto reload while any changes happens to ConfigMap data. This is handled via
[kloader](https://github.com/appscode/kloader). Voyager keeps the ingress resource and the configuration in sync
by performing processing on the resources.

Certificate watcher watch and process certificates third party data and obtain a ACME certificates.


### Third Party Resources
`voyager` depends on two Third Party Resource Object `ingress.appscode.com` and `certificate.appscode.com`. Those two objects
can be created using following data.

```yaml
metadata:
  name: ingress.appscode.com
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "Extended ingress support for Kubernetes by appscode.com"
versions:
  - name: v1beta1
```

```yaml
metadata:
  name: certificate.appscode.com
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "A specification of a Let's Encrypt Certificate to manage."
versions:
  - name: v1beta1
```

```sh
# Create Third Party Resource
$ kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/ingress.yaml
$ kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/certificate.yaml
```
