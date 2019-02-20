---
title: Placement of Ingress Pods | Voyager
menu:
  product_voyager_9.0.0:
    identifier: pod-placement-ingress
    name: Pod Placement
    parent: ingress-guides
    weight: 50
product_name: voyager
menu_name: product_voyager_9.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Placement of Ingress Pods

Voyager has rich support for how HAProxy pods are placed on cluster nodes. Please check [here](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) to understand Kubernetes' support for pod placement.

## Before You Begin

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using [Minikube](https://github.com/kubernetes/minikube).

Now, install Voyager operator in your `minikube` cluster following the steps [here](/docs/setup/install.md).

```console
minikube start
# install without RBAC
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/9.0.0/hack/deploy/voyager.sh \
  | bash -s -- minikube
```

To keep things isolated, this tutorial uses a separate namespace called `demo` throughout this tutorial. Run the following command to prepare your cluster for this tutorial:

```console
$ curl -fSsL https://raw.githubusercontent.com/appscode/voyager/9.0.0/docs/examples/ingress/pod-placement/deploy-servers.sh | bash
+ kubectl create namespace demo
namespace "demo" created
+ kubectl run nginx --image=nginx --namespace=demo
deployment "nginx" created
+ kubectl expose deployment nginx --name=web --namespace=demo --port=80 --target-port=80
service "web" exposed
+ kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4 --namespace=demo
deployment "echoserver" created
+ kubectl expose deployment echoserver --name=rest --namespace=demo --port=80 --target-port=8080
service "rest" exposed
```

### Choosing Workload Kind

By default Voyager will run HAProxy pods using `Deployment`. Since 8.0.1 release, Voyager can run HAProxy pods using either Deployment or DaemonSet. Set the annotation `ingress.appscode.com/workload-kind` on an ingress object to either `Deployment` or `DaemonSet` to enable this feature. If this annotation is missing, HAProxy pods will be run using a `Deployment` as before.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: ingress-w-node-selector
  namespace: demo
  annotations:
    ingress.appscode.com/workload-kind: DaemonSet
```

### Using Node Selector

[Node selectors](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector) can be used assign HAProxy ingress pods to specific nodes. Below is an example where ingress pods are run on node with name`minikube`.

```console
kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/9.0.0/docs/examples/ingress/pod-placement/ingress-w-node-selector.yaml
```

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: ingress-w-node-selector
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
spec:
  nodeSelector:
    kubernetes.io/hostname: minikube
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
```

If you are using official `extensions/v1beta1` ingress api group, use `ingress.appscode.com/node-selector` annotation to provide the selectors. For example:

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-w-node-selector
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
    ingress.appscode.com/node-selector: '{"kubernetes.io/hostname": "minikube"}'
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
```

### Using Pod Anti-affinity

[Affinity rules](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) can be used assign HAProxy ingress pods to specific nodes or ensure that 2 separate HAProxy ingress pods are not placed on same node. Affinity rules are set via `spec.affinity` field in Voyager Ingress CRD. Below is an example where ingress pods are spread over run on node with name`minikube`.

```console
kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/9.0.0/docs/examples/ingress/pod-placement/ingress-w-pod-anti-affinity.yaml
```

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: ingress-w-pod-anti-affinity
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: origin
            operator: In
            values:
            - voyager
          - key: origin-name
            operator: In
            values:
            - voyager-ingress-w-pod-anti-affinity
        topologyKey: 'kubernetes.io/hostname'
```

### Using Taints and Toleration

Using [taints and toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/), you can run voyager pods on dedicated nodes.

```console
# taint nodes where only HAProxy ingress pods will run
kubectl taint nodes minikube IngressOnly=true:NoSchedule

kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/9.0.0/docs/examples/ingress/pod-placement/ingress-w-toleration.yaml
```

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: ingress-w-toleration
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
  tolerations:
  - key: IngressOnly
    operator: Equal
    value: 'true'
    effect: NoSchedule
```

If you are using official `extensions/v1beta1` ingress api group, use `ingress.appscode.com/tolerations` annotation to provide the toleration information. For example:

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-w-toleration
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
    ingress.appscode.com/tolerations: '[{"key": "IngressOnly", "operator": "Equal", "value": "true", "effect": "NoSchedule"}]'
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
```

You can use these various option in combination with each other to achieve desired result. Say, you want to run your HAProxy pods on master instances. This can be done using an Ingress like below:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: ingress-w-node-selector
  namespace: demo
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: 'true'
    ingress.appscode.com/replicas: '2'
spec:
  nodeSelector:
    node-role.kubernetes.io/master: ""
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: rest
          servicePort: 80
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
```
