---
title: Ingress Class | Voyager
description: Ingress Class
menu:
  product_voyager_5.0.0-rc.10:
    identifier: ingress-class
    name: Ingress Class
    parent: concepts
    weight: 40
product_name: voyager
menu_name: product_voyager_5.0.0-rc.10
section_menu_id: concepts
---

# Running voyager alongside with other ingress controller

Voyager can be configured to handle default kubernetes ingress or only ingress.appscode.com. voyager can also be run
along side with other controllers.

```console
  --ingress-class
  // this flag can be set to 'voyager' to handle only ingress
  // with annotation kubernetes.io/ingress.class=voyager.

  // If unset, voyager will also handle ingress without ingress-class annotation.
```
