---
menu:
  product_voyager_5.0.0-rc.7:
    name: Annotations
    parent: ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_5.0.0-rc.7
section_menu_id: tutorials
---

## Custom Annotations to LoadBalancer Service or Pods

If the LoadBalancer service and Pods needs to be set custom annotations, those can be
set via two ingress options.

`ingress.appscode.com/annotations-service`
Json encoded annotations map that will be applied to LoadBalancer service.

ie.
```
ingress.appscode.com/annotations-service = {"foo": "bar", "service-annotation": "set"}
```
This will add the `foo:bar` and `service-annotation:set` to the Service annotation.


`ingress.appscode.com/annotations-pod`
Json encoded annotations map that will be applied to LoadBalancer pods.

ie.
```
ingress.appscode.com/annotations-pod = {"foo": "bar", "pod-annotation": "set"}
```
This will add the `foo:bar` and `pod-annotation:set` to all the pods' annotation.
