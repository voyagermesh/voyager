## Specify NodePort

If you are using a `NodePort` or `LoadBalancer` type Ingress, a `NodePort` or `LoadBalancer` type Service is used to expose HAProxy pods respectively. If no node port is specified for each HAProxy Service port, Kubernetes will randomly assign one for you.

Since 3.2.0, you have the option to specify a NodePort for each HAProxy Service port. This allows you to guarantee that the port will not get changed, as you make changes to an Ingress object. If you specify nothing, Kubernetes will auto assign as before.

Below is an example Ingress that demonstrates this feature:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
spec:
  rules:
  - host: one.example.com
    http:
      port: '8989'
      nodePort: '32666'
      paths:
      - path: /t1
        backend:
          serviceName: test-service
          servicePort: '80'
      - path: /t2
        backend:
          serviceName: test-service
          servicePort: '80'
  - host: other.example.com
    http:
      port: '8989'
      nodePort: '32666'
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  - host: appscode.example.com
    tcp:
      port: '4343'
      nodePort: '32667'
      backend:
        serviceName: test-service
        servicePort: '80'
```

Since no `ingress.appscode.com/type` annotation is used, this Ingress is going to expose HAProxy pods via a `LoadBalancer` service. To use a `NodePort` type Ingress, add this annotation `ingress.appscode.com/type: NodePort`.


```yaml
$ kubectl get svc voyager-test-ingress -o yaml

apiVersion: v1
kind: Service
metadata:
  annotations:
    ingress.appscode.com/origin-api-schema: voyager.appscode.com/v1beta1
    ingress.appscode.com/origin-name: test-ingress
  creationTimestamp: 2017-09-07T12:01:35Z
  name: voyager-test-ingress
  namespace: default
  resourceVersion: "1161"
  selfLink: /api/v1/namespaces/default/services/voyager-test-ingress
  uid: 4baf3dbf-93c4-11e7-bf12-080027a9d54f
spec:
  clusterIP: 10.0.0.41
  externalTrafficPolicy: Cluster
  ports:
  - name: tcp-8989
    nodePort: 32666
    port: 8989
    protocol: TCP
    targetPort: 8989
  - name: tcp-4343
    nodePort: 32667
    port: 4343
    protocol: TCP
    targetPort: 4343
  selector:
    origin: voyager
    origin-api-group: voyager.appscode.com
    origin-api-version: v1beta1
    origin-name: test-ingress
  sessionAffinity: None
  type: NodePort
status:
  loadBalancer: {}
```


For this Ingress, a `LoadBalancer` Service will listen to `8989` and `4343` port for incoming HTTP

If you think about the Service that will be created here, will have one service port 8989 that points to container port 8989 and uses NodePort 3266
`kubectl get service voyager-test-ingress -o yaml` (edited)


connections and these port will map to specified nodeports, and will pass any request coming to it to the desired backend.

> For one Ingress Type you cannot have multiple rules listening to same port or same node port, even if they do not have
same `host`.


Port 8989 has 2 separate hosts appscode.example.com and other.example.com .  appscode.example.com has 2 paths
Since they all expose via the same HTTP port, they must use the same NodePort

If you think about the Service that will be created here, will have one service port 8989 that points to container port 8989 and uses NodePort 3266
`kubectl get service voyager-test-ingress -o yaml` (edited)

