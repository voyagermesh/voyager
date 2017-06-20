Voyager can use AWS certificate manager to terminate SSL connections for `LoadBalancer` type ingress in "aws" provider. To use this feature,
add the following annotations to Ingress;

```yaml
  ingress.appscode.com/annotations-service: |
    {
      "service.beta.kubernetes.io/aws-load-balancer-ssl-cert": "arn:aws:acm:..."
      "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "http",
    }
```

Voyager operator will apply these annotation on `LoadBalancer` service used to expose HAProxy to internet. This service will (logically) listen on port 443, terminate SSL and forward to port 80 on HAProxy pods. Also, ELB will listen on port 80 and forward cleartext traffic to port 80.

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
  - port: 80
    targetPort: 80
   ...
```
[Elastic Load Balancing](http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/x-forwarded-headers.html#x-forwarded-proto) stores the protocol used between the client and the load balancer in the `X-Forwarded-Proto` request header and passes the header along to HAProxy. The `X-Forwarded-Proto` request header helps HAProxy identify the protocol (HTTP or HTTPS) that a client used to connect to load balancer. If you would like to redirect cleartext client traffic on port 80 to port 443, please add redirect backend rules when `X-Forwarded-Proto` header value is `HTTPS`. Please see example rules here: https://www.exratione.com/2014/10/managing-haproxy-configuration-when-your-server-may-or-may-not-be-behind-an-ssl-terminating-proxy/
