---
title: Uninstall Voyager
description: Uninstallation guide for Voyager
menu:
  docs_{{ .version }}:
    identifier: uninstall-voyager-enterprise
    name: Voyager
    parent: uninstallation-guide
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: setup
---

# Uninstall Voyager

To uninstall Voyager, run the following command:

<ul class="nav nav-tabs" id="installerTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link active" id="helm3-tab" data-toggle="tab" href="#helm3" role="tab" aria-controls="helm3" aria-selected="true">Helm 3</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="script-tab" data-toggle="tab" href="#script" role="tab" aria-controls="script" aria-selected="false">YAML</a>
  </li>
</ul>
<div class="tab-content" id="installerTabContent">
  <div class="tab-pane fade show active" id="helm3" role="tabpanel" aria-labelledby="helm3-tab">

## Using Helm 3

In Helm 3, release names are [scoped to a namespace](https://v3.helm.sh/docs/faq/#release-names-are-now-scoped-to-the-namespace). So, provide the namespace you used to install the operator when installing.

```bash
$ helm uninstall voyager --namespace voyager
```

</div>
<div class="tab-pane fade" id="script" role="tabpanel" aria-labelledby="script-tab">

## Using YAML (with helm 3)

If you prefer to not use Helm, you can generate YAMLs from Voyager chart and uninstall using `kubectl`.

```bash
$ helm template voyager oci://ghcr.io/appscode-charts/voyager \
  --namespace voyager | kubectl delete -f -
```

</div>
</div>
