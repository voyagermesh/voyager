---
title: Upgrade | Voyager
description: Voyager Upgrade
menu:
  docs_{{ .version }}:
    identifier: upgrade-voyager
    name: Upgrade
    parent: setup
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: setup
---

# Upgrading Voyager

This guide will show you how to upgrade various Voyager components. Here, we are going to show how to upgrade from an old Voyager version to the new version, how to migrate between the enterprise edition and community edition, and how to update the license, etc.

## Upgrading Voyager from `v2021.x.x` to `{{< param "info.version" >}}`

In order to upgrade from Voyager `v2021.x.x` to `{{< param "info.version" >}}`, please follow the following steps.

{{< notice type="warning" message="Please note that since v2021.09.16, we recommend installing Voyager operator in the `voyager` namespace. The upgrade instructions on this page assumes so. If you have currently installed the operator in a different namespace like `kube-system`, either follow the instructions with appropriate updates, or first uninstall the existing operator and then reinstall in the `voyager` namespace." >}}

#### 1. Update Voyager Catalog CRDs

Helm [does not upgrade the CRDs](https://github.com/helm/helm/issues/6581) bundled in a Helm chart if the CRDs already exist. So, to upgrade the Voyager catalog CRD, please run the command below:

```bash
kubectl apply -f https://github.com/voyagermesh/installer/raw/{{< param "info.version" >}}/crds/voyager-crds.yaml
```

#### 2. Upgrade Voyager Operator

Now, upgrade the Voyager helm chart using the following command. You can find the latest installation guide [here](/docs/setup/README.md).

```bash
helm upgrade voyager-operator appscode/voyager \
  --version {{< param "info.version" >}} \
  --namespace voyager \
  --set cloudProvider=$provider \
  --set-file license=/path/to/the/license.txt
```

## Migration Between Community Edition and Enterprise Edition

Voyager supports seamless migration between community edition and enterprise edition. You can run the following commands to migrate between them.

<ul class="nav nav-tabs" id="migrationTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link active" id="mgr-helm3-tab" data-toggle="tab" href="#mgr-helm3" role="tab" aria-controls="mgr-helm3" aria-selected="true">Helm 3</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="mgr-yaml-tab" data-toggle="tab" href="#mgr-yaml" role="tab" aria-controls="mgr-yaml" aria-selected="false">YAML</a>
  </li>
</ul>
<div class="tab-content" id="migrationTabContent">
  <div class="tab-pane fade show active" id="mgr-helm3" role="tabpanel" aria-labelledby="mgr-helm3">

#### Using Helm 3

**From Community Edition to Enterprise Edition:**

In order to migrate from Voyager community edition to Voyager enterprise edition, please run the following command,

```bash
helm upgrade voyager-operator appscode/voyager \
  --namespace voyager \
  --reuse-values \
  --set-file license=/path/to/voyager-enterprise-license.txt
```

**From Enterprise Edition to Community Edition:**

In order to migrate from Voyager enterprise edition to Voyager community edition, please run the following command,

```bash
helm upgrade voyager-operator appscode/voyager \
  --namespace voyager \
  --reuse-values \
  --set-file license=/path/to/voyager-community-license.txt
```

</div>
<div class="tab-pane fade" id="mgr-yaml" role="tabpanel" aria-labelledby="mgr-yaml">

**Using YAML (with helm 3)**

**From Community Edition to Enterprise Edition:**

In order to migrate from Voyager community edition to Voyager enterprise edition, please run the following command,

```bash
# Install Voyager enterprise edition
helm template voyager appscode/voyager \
  --namespace voyager --create-namespace \
  --version {{< param "info.version" >}} \
  --set cleaner.skip=true \
  --set-file license=/path/to/voyager-enterprise-license.txt | kubectl apply -f -
```

**From Enterprise Edition to Community Edition:**

In order to migrate from Voyager enterprise edition to Voyager community edition, please run the following command,

```bash
# Install Voyager community edition
helm template voyager appscode/voyager \
  --namespace voyager --create-namespace \
  --version {{< param "info.version" >}} \
  --set cleaner.skip=true \
  --set-file license=/path/to/voyager-community-license.txt | kubectl apply -f -
```

</div>
</div>

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
    <a class="nav-link" id="lu-yaml-tab" data-toggle="tab" href="#lu-yaml" role="tab" aria-controls="lu-yaml" aria-selected="false">YAML</a>
  </li>
</ul>
<div class="tab-content" id="luTabContent">
  <div class="tab-pane fade show active" id="lu-helm3" role="tabpanel" aria-labelledby="lu-helm3">

#### Using Helm 3

```bash
helm upgrade voyager-operator appscode/voyager \
  --namespace voyager \
  --reuse-values \
  --set-file license=/path/to/new/license.txt
```

</div>
<div class="tab-pane fade" id="lu-yaml" role="tabpanel" aria-labelledby="lu-yaml">

#### Using YAML (with helm 3)

```bash
helm template voyager appscode/voyager \
  --namespace voyager --create-namespace \
  --set cleaner.skip=true \
  --set-file license=/path/to/new/license.txt | kubectl apply -f -
```

</div>
</div>
