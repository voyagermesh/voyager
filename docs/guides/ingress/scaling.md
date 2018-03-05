---
title: Scaling Ingress | Voyager
menu:
  product_voyager_6.0.0-rc.2:
    identifier: scaling-ingress
    name: Scaling Ingress
    parent: ingress-guides
    weight: 45
product_name: voyager
menu_name: product_voyager_6.0.0-rc.2
section_menu_id: guides
---

# Scaling Ingress

## Replicas

For each Ingress resource, Voyager deploys HAProxy in a Deployment prefixed by
`voyager-` and the name of the Ingress.

This Deployment has `.spec.replicas = 1` by default. To change the desired
number of replicas, use the `ingress.appscode.com/replicas` annotation.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: my-app
  annotations:
    ingress.appscode.com/replicas: '2'
spec:
  backend:
    serviceName: my-app
    servicePort: '80'
```

```console
$ kubectl get deploy voyager-my-app
NAME               DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
voyager-my-app     2         2         2            2           1d
```

## Horizontal Pod Autoscaling

[Kubernetes has the HorizontalPodAutoscaler object for autoscaling pods](https://kubernetes.io/docs/guides/run-application/horizontal-pod-autoscale/).

> With Horizontal Pod Autoscaling, Kubernetes automatically scales the number
> of pods in a replication controller, deployment or replica set based on
> observed CPU utilization (or, with alpha support, on some other, application-provided metrics).

To set up a HorizontalPodAutoscaler for a Voyager HAPRoxy deployment, you can
use the `kubectl autoscale` command or defining a HorizontalPodAutoscaler
resource.

```console
kubectl autoscale deployment voyager-my-app --cpu-percent=20 --min=2 --max=10
```

```yaml
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: voyager-my-app
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta1
    kind: Deployment
    name: voyager-my-app
  minReplicas: 2
  maxReplicas: 10
targetCPUUtilizationPercentage: 20%
```

```
$ kubectl get hpa
NAME                  REFERENCE                        TARGETS    MINPODS   MAXPODS   REPLICAS   AGE
voyager-my-app        Deployment/voyager-my-app        0% / 20%   2         10        2          1d
```

## Node Autoscaling
If you are using [autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler) to dynamically add and remove nodes, you might be interested in using `ingress.appscode.com/node-selector` to control which hosts are selected to run HAProxy pods. This is a recommended annotation for HostPort type ingress.
