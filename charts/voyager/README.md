# Voyager

[Voyager by AppsCode](https://github.com/appscode/voyager) - Secure HAProxy Ingress Controller for Kubernetes

## TL;DR;

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm install voyager-operator appscode/voyager -n kube-system
```

## Introduction

This chart bootstraps an [ingress controller](https://github.com/appscode/voyager) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.12+

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install my-release appscode/voyager -n kube-system
```

The command deploys Voyager Controller on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release`:

```console
$ helm delete my-release -n kube-system
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the Voyager chart and their default values.


| Parameter                            | Description                                                   | Default               |
| ------------------------------------ | ------------------------------------------------------------- | ----------------------|
| `replicaCount`                       | Number of operator replicas to create (only 1 is supported)   | `1`                   |
| `voyager.registry`                   | Docker registry used to pull Voyager image                    | `appscode`            |
| `voyager.repository`                 | Voyager container image                                       | `voyager`             |
| `voyager.tag`                        | Voyager container image tag                                   | `v12.0.0-rc.0`             |
| `haproxy.registry`                   | Docker registry used to pull HAProxy image                    | `appscode`            |
| `haproxy.repository`                 | HAProxy container image                                       | `haproxy`             |
| `haproxy.tag`                        | HAProxy container image tag                                   | `1.9.6-v12.0.0-rc.0-alpine` |
| `cleaner.registry`                   | Docker registry used to pull Webhook cleaner image            | `appscode`            |
| `cleaner.repository`                 | Webhook cleaner container image                               | `kubectl`             |
| `cleaner.tag`                        | Webhook cleaner container image tag                           | `v1.11`               |
| `imagePullSecrets`                   | Specify image pull secrets                                    | `[]`                  |
| `imagePullPolicy`                    | Image pull policy                                             | `IfNotPresent`        |
| `cloudProvider`                      | Name of cloud provider                                        | `nil`                 |
| `cloudConfig`                        | Path to cloud config                                          | ``                    |
| `criticalAddon`                      | If true, installs Voyager operator as critical addon          | `false`               |
| `logLevel`                           | Log level for operator                                        | `3`                   |
| `persistence.enabled`                | Enable mounting cloud config                                  | `false`               |
| `persistence.hostPath`               | Host mount path for cloud config                              | `/etc/kubernetes`     |
| `affinity`                           | Affinity rules for pod assignment                             | `{}`                  |
| `annotations`                        | Annotations applied to operator pod(s)                        | `{}`                  |
| `nodeSelector`                       | Node labels for pod assignment                                | `{}`                  |
| `tolerations`                        | Tolerations used pod assignment                               | `{}`                  |
| `rbac.create`                        | If `true`, create and use RBAC resources                      | `true`                |
| `serviceAccount.create`              | If `true`, create a new service account                       | `true`                |
| `serviceAccount.name`                | Service account to be used. If not set and `serviceAccount.create` is `true`, a name is generated using the fullname template | `` |
| `ingressClass`                       | Ingress class to watch for. If empty, it handles all ingress  | ``                    |
| `templates.cfgmap`                   | Name of configmap with custom templates                       | ``                    |
| `apiserver.groupPriorityMinimum`     | The minimum priority the group should have.                   | 10000                 |
| `apiserver.versionPriority`          | The ordering of this API inside of the group.                 | 15                    |
| `apiserver.enableValidatingWebhook`  | Configure apiserver as admission webhooks for Voyager CRDs     | `true`                |
| `apiserver.ca`                       | CA certificate used by main Kubernetes api server             | `not-ca-cert`         |
| `apiserver.bypassValidatingWebhookXray` | If true, bypasses validating webhook xray checks           | `false`               |
| `apiserver.useKubeapiserverFqdnForAks`  | If true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 | `true`             |
| `apiserver.healthcheck.enabled`         | Enable readiness and liveliness probes                     | `false`               |
| `enableAnalytics`                       | Send usage events to Google Analytics                      | `true`                |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
$ helm install my-release appscode/voyager -n kube-system --set image.tag=v0.2.1
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```console
$ helm install my-release appscode/voyager -n kube-system --values values.yaml
```
