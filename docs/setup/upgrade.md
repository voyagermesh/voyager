---
title: Upgrade Voyager
description: Voyager Upgrade
menu:
  docs_{{ .version }}:
    identifier: upgrade-voyager
    name: Upgrade
    parent: setup
    weight: 18
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: setup
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Upgrading Voyager

This guide will show you how to upgrade various Voyager components. Here, we are going to show how to upgrade from an old Voyager version to the new version, and how to update the license, etc.

## Upgrading Voyager from `v12.0.0` or prior to `v2021.04.24-rc.0`

In order to upgrade from Voyager `v12.0.0` to `v2021.04.6`, please follow the following steps.

#### 1. Update Voyager CRDs

Helm [does not upgrade the CRDs](https://github.com/helm/helm/issues/6581) bundled in a Helm chart if the CRDs already exist. So, to upgrde the Voyager CRDs, please run the command below:

```bash
kubectl apply -f https://github.com/voyagermesh/installer/raw/{{< param "info.version" >}}/voyager-crds.yaml
```

#### 2. Upgrade Voyager Operator

Now, upgrade the Voyager helm chart using the following command. You can find the latest installation guide [here](/docs/setup/README.md).

```bash
helm upgrade voyager-operator -n kube-system appscode/voyager \
  --reuse-values \
  --version {{< param "info.version" >}}
```


## Updating License

Voyager support updating license without requiring any re-installation. Voyager creates a Secret named `<helm release name>-license` with the license file. You just need to update the Secret. The changes will propagate automatically to the operator and it will use the updated license going forward.

Follow the below instructions to update the license:

- Get a new license and save it into a file.
- Then, run the following upgrade command based on your installation.

<ul class="nav nav-tabs" id="luTabs" role="tablist">
  <li class="nav-item">
    <a class="nav-link active" id="lu-helm3-tab" data-toggle="tab" href="#lu-helm3" role="tab" aria-controls="lu-helm3" aria-selected="true">Helm 3</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="lu-helm2-tab" data-toggle="tab" href="#lu-helm2" role="tab" aria-controls="lu-helm2" aria-selected="false">Helm 2</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="lu-yaml-tab" data-toggle="tab" href="#lu-yaml" role="tab" aria-controls="lu-yaml" aria-selected="false">YAML</a>
  </li>
</ul>
<div class="tab-content" id="luTabContent">
  <div class="tab-pane fade show active" id="lu-helm3" role="tabpanel" aria-labelledby="lu-helm3">

#### Using Helm 3

```bash
helm upgrade voyager-operator -n kube-system appscode/voyager \
  --reuse-values \
  --set-file license=/path/to/new/license.txt
```

</div>
<div class="tab-pane fade" id="lu-helm2" role="tabpanel" aria-labelledby="lu-helm2">

#### Using Helm 2

```bash
helm upgrade voyager-operator appscode/voyager \
  --reuse-values \
  --set-file license=/path/to/new/license.txt
```

</div>
<div class="tab-pane fade" id="lu-yaml" role="tabpanel" aria-labelledby="lu-yaml">

#### Using YAML (with helm 3)

```bash
helm template voyager-operator -n kube-system appscode/voyager \
  --set cleaner.skip=true \
  --set-file license=/path/to/new/license.txt | kubectl apply -f -
```

</div>
</div>
