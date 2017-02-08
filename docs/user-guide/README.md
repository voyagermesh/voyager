# User Guide
This guide will walk you through deploying the voyager controller.

## High Level Tasks
* Create `ingress.appscode.com` and `certificate.appscode.com` Third Party Resource
* Create voyager Deployment

## Deploying voyager

#### Create the Third Party Resources
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



