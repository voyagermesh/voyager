# Installation Guide

## Using YAML
Voyager can be installed using YAML files includes in the [/hack/deploy](/hack/deploy) folder.

```console
$ export CLOUD_PROVIDER=<provider-name> # ie:
                                        # - gce
                                        # - gke
                                        # - aws
                                        # - azure
                                        # - acs (aka, Azure Container Service)

$ export CLOUD_CONFIG=<path>            # The path to the cloud provider configuration file.
                                        # Empty string for no configuration file.
                                        # Leave it empty for `gce`, `gke`, `aws` and bare metal clusters.
                                        # For azure/acs, use `/etc/kubernetes/azure.json`.
                                        # This file was created during the cluster provisioning process.
                                        # Voyager uses this to connect to cloud provider api.

# Install without RBAC roles
$ curl https://raw.githubusercontent.com/appscode/voyager/3.2.0-rc.0/hack/deploy/without-rbac.yaml \
    | envsubst \
    | kubectl apply -f -

# Install with RBAC roles
$ curl https://raw.githubusercontent.com/appscode/voyager/3.2.0-rc.0/hack/deploy/with-rbac.yaml \
    | envsubst \
    | kubectl apply -f -
```

There are cloud provider specific installer scripts available in [/hack/deploy](/hack/deploy) folder. To use in a RBAC enabled cluster, pass the `--rbac` flag.
```console
# Deploy in minikube
$ ./hack/deploy/minikube.sh [--rbac]

# Deploy in Amazon AWS EC2
$ ./hack/deploy/aws.sh [--rbac]

# Deploy in Google Compute Cloud(GCE)
$ ./hack/deploy/gce.sh [--rbac]

# Deploy in Google Container Engine(GKE)
$ ./hack/deploy/gke.sh [--rbac]

# Deploy in Microsoft Azure
$ ./hack/deploy/azure.sh [--rbac]

# Deploy in Azure Container Service(ACS)
$ ./hack/deploy/acs.sh [--rbac]

# Deploy in Baremetal providers
$ ./hack/deploy/baremetal.sh [--rbac]
```


## Using Helm
Voyager can be installed via [Helm](https://helm.sh/) using the [chart](/chart/voyager) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```console
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

Now, you are ready to create your first ingress using Voyager.

## Using kubectl
Since Voyager uses its own TPR/CRD, you need to use full resource kind to find it with kubectl.
```console
$ kubectl get ingress.voyager.appscode.com --all-namespaces
```
