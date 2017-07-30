> New to Voyager? Please start [here](/docs/tutorial.md).

# Installation Guide

## Using YAML
Voyager can be installed using YAML files includes in the [/hack/deploy](/hack/deploy) folder.

```console
# Install without RBAC roles
$ curl https://raw.githubusercontent.com/appscode/voyager/3.1.1/hack/deploy/without-rbac.yaml \
  | kubectl apply -f -


# Install with RBAC roles
$ curl https://raw.githubusercontent.com/appscode/voyager/3.1.1/hack/deploy/with-rbac.yaml \
  | kubectl apply -f -
```

## Using Helm
Voyager can be installed via [Helm](https://helm.sh/) using the [chart](/chart/voyager) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```bash
$ helm install chart/voyager --name my-release
```
To see the detailed configuration options, visit [here](/chart/voyager/README.md).


## Verify installation
To check if Voyager operator pods have started, run the following command:
```console
$ kubectl get pods --all-namespaces -l app=voyager --watch
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

Now, to confirm TPR groups have been registered by the operator, run the following command:
```console
$ kubectl get thirdpartyresources -l app=voyager
```

Now, you are ready to [take your first backup](/docs/tutorial.md) using Voyager.
