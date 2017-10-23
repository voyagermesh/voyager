```console
$ kubectl create -f ./docs/examples/monitoring/demo-0.yaml
namespace "demo" created
deployment "prometheus-operator" created

$ kubectl get pods -n demo
NAME                                  READY     STATUS    RESTARTS   AGE
prometheus-operator-449376836-mq4p3   1/1       Running   0          1m
```

```console
$ kubectl create -f ./docs/examples/monitoring/demo-1.yaml
prometheus "prometheus" created
service "prometheus" created

$ kubectl get pods -n demo --watch
NAME                                  READY     STATUS    RESTARTS   AGE
prometheus-operator-449376836-mq4p3   1/1       Running   0          1m
prometheus-prometheus-0   0/2       ContainerCreating   0         6s
prometheus-prometheus-0   1/2       Running   0         25s
prometheus-prometheus-0   2/2       Running   0         26s
^C⏎
```

```console
$ ./hack/deploy/minikube.sh

$ kubectl get pods -l app=voyager --all-namespaces --watch
NAMESPACE     NAME                                READY     STATUS    RESTARTS   AGE
kube-system   voyager-operator-2464855905-gfdlq   1/1       Running   0          22s
^C⏎
```

```console
$ sudo nano /etc/hosts

127.0.0.1       localhost
127.0.1.1       beast
192.168.99.100       voyager.demo
```










```
~/g/s/g/a/v/h/deploy (d2) $ kubectl get svc -n demo
NAME                         CLUSTER-IP   EXTERNAL-IP   PORT(S)               AGE
prometheus                   10.0.0.142   <pending>     9090:30900/TCP        37m
prometheus-operated          None         <none>        9090/TCP              37m
test-server                  10.0.0.28    <none>        80/TCP                9m
voyager-test-ingress         10.0.0.81    <pending>     80:30446/TCP          9m
voyager-test-ingress-stats   10.0.0.36    <none>        56789/TCP,56790/TCP   6s
~/g/s/g/a/v/h/deploy (d2) $ 
~/g/s/g/a/v/h/deploy (d2) $ kubectl get servicemonitor -n demo
NAME                        KIND
voyager-demo-test-ingress   ServiceMonitor.v1alpha1.monitoring.coreos.com
~/g/s/g/a/v/h/deploy (d2) $ kubectl get servicemonitor -n demo
NAME                        KIND
voyager-demo-test-ingress   ServiceMonitor.v1alpha1.monitoring.coreos.com
~/g/s/g/a/v/h/deploy (d2) $ 
~/g/s/g/a/v/h/deploy (d2) $ 
~/g/s/g/a/v/h/deploy (d2) $ kubectl get servicemonitor -n demo -o yaml
apiVersion: v1
items:
- apiVersion: monitoring.coreos.com/v1alpha1
  kind: ServiceMonitor
  metadata:
    creationTimestamp: 2017-07-21T00:31:06Z
    labels:
      app: voyager
    name: voyager-demo-test-ingress
    namespace: demo
    resourceVersion: "4118"
    selfLink: /apis/monitoring.coreos.com/v1alpha1/namespaces/demo/servicemonitors/voyager-demo-test-ingress
    uid: e2bc3888-6dab-11e7-87c7-080027820ec8
  spec:
    endpoints:
    - path: /voyager.appscode.com/v1beta1/namespaces/demo/ingresses/test-ingress/metrics
      port: http
      targetPort: 0
    namespaceSelector:
      matchNames:
      - demo
    selector:
      matchLabels:
        feature: stats
        origin: voyager
        origin-api-group: voyager.appscode.com
        origin-api-version: v1beta1
        origin-name: test-ingress
kind: List
metadata: {}
resourceVersion: ""
selfLink: ""
~/g/s/g/a/v/h/deploy (d2) $ kubectl get svc voyager-test-ingress-stats -n demo -o yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    ingress.appscode.com/origin-api-schema: voyager.appscode.com/v1beta1
    ingress.appscode.com/origin-name: test-ingress
  creationTimestamp: 2017-07-21T00:31:16Z
  labels:
    feature: stats
    origin: voyager
    origin-api-group: voyager.appscode.com
    origin-api-version: v1beta1
    origin-name: test-ingress
  name: voyager-test-ingress-stats
  namespace: demo
  resourceVersion: "4182"
  selfLink: /api/v1/namespaces/demo/services/voyager-test-ingress-stats
  uid: e8c26919-6dab-11e7-87c7-080027820ec8
spec:
  clusterIP: 10.0.0.36
  ports:
  - name: stats
    port: 56789
    protocol: TCP
    targetPort: stats
  - name: http
    port: 56790
    protocol: TCP
    targetPort: http
  selector:
    origin: voyager
    origin-api-group: voyager.appscode.com
    origin-api-version: v1beta1
    origin-name: test-ingress
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
~/g/s/g/a/v/h/deploy (d2) $ 
```























































```console
$ kubectl delete deployment voyager-operator -n kube-system
deployment "voyager-operator" deleted
$ kubectl delete svc voyager-operator -n kube-system
service "voyager-operator" deleted
```
