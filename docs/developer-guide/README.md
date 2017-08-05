## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Voyager.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Voyager should not change.

### Build Voyager
Some of the Voyager development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

#### Setup GO
Voyager is written in Google's GO programming language. Currently, Voyager is developed and tested on **go 1.8.3**. If you haven't set up a GO
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install GO.

#### Download Source

```console
$ go get github.com/appscode/voyager
$ cd $(go env GOPATH)/src/github.com/appscode/voyager
```

#### Install Dev tools
To install various dev tools for Voyager, run the following command:
```console
$ ./hack/builddeps.sh
```

#### Build Binary
```
$ ./hack/make.py
$ voyager version
```

#### Dependency management
Voyager uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. Dependencies are already checked in the `vendor` folder.
If you want to update/add dependencies, run:
```console
$ glide slow
```

#### Build Docker images
To build and push your custom Docker image, follow the steps below. To release a new version of Voyager, please follow the [release guide](/docs/developer-guide/release.md).

```console
# Build Docker image
$ ./hack/docker/voyager/setup.sh

# Add docker tag for your repository
$ docker tag appscode/voyager:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>
```

#### Build HAProxy
We package HAProxy and [Kloader](https://github.com/appscode/kloader) into a Ubuntu 16.04 based Docker image.
```console
$ ./hack/docker/haproxy/<version>/setup.sh
$ ./hack/docker/haproxy/<version>/setup.sh push
```

#### Generate CLI Reference Docs
```console
$ ./hack/gendocs/make.sh 
```

### Run Test
#### Run Short Unit Test by running
```console
go test -v ./cmd/... ./pkg/...
```

#### Run Full Test
To Run Full unit test You need to provide some secret in `hack/configs/.env` file. Or You may add them as
environment variables.
```console
TEST_GCE_SERVICE_ACCOUNT_DATA
TEST_GCE_PROJECT
TEST_ACME_USER_EMAIL
TEST_DNS_DOMAINS
```
Then run
```console
$ ./hack/make.py test unit
```

#### Run e2e Test
```
$ ./hack/make.py test minikube # Run Test against minikube, this requires minikube to be set up and started.

$ ./hack/make.py test e2e -cloud-provider=gce # Test e2e against gce cluster

$ ./hack/make.py test integration -cloud-provider=gce # Run Integration test against gce
                                                      # This requires voyager to be deployed in the cluster.

```

```
- Run only one e2e test
$ ./hack/make.py test e2e -cloud-provider=gce -test-only=CoreIngress


- Run One test but do not delete all resource that are created
$ ./hack/make.py test minikube -cloud-provider=gce -test-only=CoreIngress -cleanup=false


- Run Service IP Persist test with provided IP
$ ./hack/make.py test e2e -cloud-provider=gce -test-only=CreateIPPersist -lb-ip=35.184.104.215

```

Tests are run only in namespaces prefixed with `test-`. So, to run tests in your desired namespace, follow these steps:
```
# create a Kubernetes namespace in minikube with
kubectl create ns test-<any-name-you-want>

# run tests
./hack/make.py test minikube -namespace test-<any-name-you-want> -max-test=1
```

<br>
## Architecture
Voyager works by implementing third party resource data watcher for kubernetes. It connects with k8s apiserver
for specific events as ADD, UPDATE and DELETE. and perform required operations.

Ingress watcher generates the configuration for HAProxy and stores it as a ConfigMaps and creates a RC with
specified HAProxy - that is configured with auto reload while any changes happens to ConfigMap data. This is handled via
[kloader](https://github.com/appscode/kloader). Voyager keeps the ingress resource and the configuration in sync
by performing processing on the resources.

Certificate watcher watch and process certificates third party data and obtain a ACME certificates.


### Third Party Resources
`voyager` depends on two Third Party Resource Object `ingress.voyager.appscode.com` and `certificate.voyager.appscode.com`. Those two objects
can be created using following data.

```yaml
metadata:
  name: ingress.voyager.appscode.com
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "Extended ingress support for Kubernetes by AppsCode"
versions:
  - name: v1beta1
```

```yaml
metadata:
  name: certificate.voyager.appscode.com
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "A specification of a Let's Encrypt Certificate to manage."
versions:
  - name: v1beta1
```

```console
# Create Third Party Resources
$ kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/3.1.2/api/extensions/tprs.yaml
```
