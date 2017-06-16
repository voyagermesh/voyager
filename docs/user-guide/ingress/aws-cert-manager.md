Voyager can use AWS certificate manager to terminate SSL connections for `LoadBalancer` type ingress in "aws" provider. To use this feature,
add the following annotations to Ingress;

```yaml
  ingress.appscode.com/annotations-service: |
    {
      "service.beta.kubernetes.io/aws-load-balancer-ssl-cert": "arn:aws:acm:..."
      "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "http",
    }
```

Voyager operator will apply these annotation on `LoadBalancer` service used to expose HAProxy to internet. This service will (logically) listen on port 443 and forward to port 80 on HAProxy pods.

```
apiVersion: v1
kind: Service
metadata:
  name: <ingress>
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: 'arn:aws:acm:...'
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: http
spec:
  type: LoadBalancer
  ports:
  - port: 443
    targetPort: 80
   ...
```
