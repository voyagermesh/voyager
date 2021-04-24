---
title: Issue Let's Encrypt certificate using HTTP-01 challenge
description: Issue Let's Encrypt certificate using HTTP-01 challenge in Kubernetes
menu:
  product_voyager_{{ .version }}:
    identifier: overview-http
    name: Overview
    parent: http-certificate
    weight: 10
product_name: voyager
menu_name: product_voyager_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Issue Let's Encrypt certificate using HTTP-01 challenge

## Deploy Voyager operator

Install Voyager operator in your cluster following the steps [here](/docs/setup/install.md).

## Create Ingress

2. We are going to use a nginx server as the backend. To deploy nginx server, run the following commands:

    ```console
    kubectl create deployment nginx --image=nginx
    kubectl expose deployment nginx --name=web --port=80 --target-port=80
    ```

3. Now create Ingress `ing.yaml`

    ```console
    kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/{{< param "info.version" >}}/docs/examples/certificate/http/ing.yaml
    ```

4. Wait for the LoadBalancer ip to be assigned. Once the IP is assigned update your DNS provider to set the LoadBlancer IP as the A record for test domain `kiteci.com`

    ```console
    $ kubectl get svc  voyager-test-ingress
    NAME                   CLUSTER-IP      EXTERNAL-IP      PORT(S)                      AGE
    voyager-test-ingress   10.39.243.239   104.198.234.66   80:32266/TCP,443:31282/TCP   19m
    ```

5. Now wait a bit for DNS to propagate. Run the following command to confirm DNS propagation.

    ```console
    $ dig +short kiteci.com
    104.198.234.66
    ```

6. Now open URL http://kiteci.com . This should show you the familiar nginx welcome page.

## Create Certificate

7. Create a secret to provide ACME user email. Change the email to a valid email address and run the following command:

    ```console
    kubectl create secret generic acme-account --from-literal=ACME_EMAIL=me@example.com
    ```

8. Create the Certificate CRD to issue TLS certificate from Let's Encrypt using HTTP challenge.

    ```console
    kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/{{< param "info.version" >}}/docs/examples/certificate/http/crt.yaml
    ```

8. Now wait a bit and you should see a new secret named `tls-kitecicom`. This contains the `tls.crt` and `tls.key`.
This secret must not have any dashes or other special characters.

    ```console
    $ kubectl get secrets
    NAME                  TYPE                                  DATA      AGE
    acme-account          Opaque                                3         20m
    default-token-zj0wv   kubernetes.io/service-account-token   3         30m
    tls-kitecicom         kubernetes.io/tls                     2         19m
    ```
    
    ```console
    $ kubectl describe cert kitecicom
    Name:		kitecicom
    Namespace:	default
    Labels:		<none>
    Annotations:	kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"voyager.appscode.com/v1beta1","kind":"Certificate","metadata":{"annotations":{},"name":"kitecicom","namespace":"default"},"spec":{"acmeU...
    API Version:	voyager.appscode.com/v1beta1
    Kind:		Certificate
    Metadata:
      Cluster Name:
      Creation Timestamp:			2017-10-29T22:07:45Z
      Deletion Grace Period Seconds:	<nil>
      Deletion Timestamp:			<nil>
      Resource Version:			1376
      Self Link:				/apis/voyager.appscode.com/v1beta1/namespaces/default/certificates/kitecicom
      UID:					97d91028-bcf5-11e7-bc3f-42010a800fd5
    Spec:
      Acme User Secret Name:	acme-account
      Challenge Provider:
        Http:
          Ingress:
            API Version:	voyager.appscode.com/v1beta1
            Kind:		Ingress
            Name:		test-ingress
      Domains:
        kiteci.com
    Events:
      FirstSeen	LastSeen	Count	From			SubObjectPath	Type		Reason		Message
      ---------	--------	-----	----			-------------	--------	------		-------
      20m		20m		1	voyager operator			Normal		IssueSuccessful	Successfully issued certificate
    ```

    If you look at the Ingress, you should see that `/.well-known/acme-challenge/` path has been added to rules. It should look like [this](/docs/examples/certificate/http/ing-with-acme-path.yaml).
    
    If you check the configmap `voyager-test-ingress`, you should see a key `haproxy.cfg` with the value similar to [this](/docs/examples/certificate/http/haproxy-with-acme.cfg).

## Update Ingress to use TLS

9. Now edit the Ingress to add `spec.tls` section.

    ```console
    $ kubectl edit ingress.voyager.appscode.com test-ingress
    
    spec:
      tls:
      - hosts:
        - kiteci.com
        ref:
          kind: Secret
          name: tls-kitecicom
    ```
    
    After editing, your Ingress should look similar to [this](/docs/examples/certificate/http/ing-tls-acme.yaml).

10. Now wait several seconds for HAProxy to reconfigure. If you check the configmap `voyager-test-ingress`, you should see a key `haproxy.cfg` with the value similar to [this](/docs/examples/certificate/http/haproxy-ssl.cfg).

    Now try the following commands:
    
    ```console
    $ curl -vv http://kiteci.com
    * Rebuilt URL to: http://kiteci.com/
    *   Trying 104.198.234.66...
    * Connected to kiteci.com (104.198.234.66) port 80 (#0)
    > GET / HTTP/1.1
    > Host: kiteci.com
    > User-Agent: curl/7.47.0
    > Accept: */*
    >
    < HTTP/1.1 301 Moved Permanently
    < Content-length: 0
    < Location: https://kiteci.com/
    <
    * Connection #0 to host kiteci.com left intact
    ```
    
    ```console
    $ curl -vv https://kiteci.com
    * Rebuilt URL to: https://kiteci.com/
    *   Trying 104.198.234.66...
    * Connected to kiteci.com (104.198.234.66) port 443 (#0)
    * found 148 certificates in /etc/ssl/certs/ca-certificates.crt
    * found 597 certificates in /etc/ssl/certs
    * ALPN, offering http/1.1
    * SSL connection using TLS1.2 / ECDHE_RSA_AES_128_GCM_SHA256
    * 	 server certificate verification OK
    * 	 server certificate status verification SKIPPED
    * 	 common name: kiteci.com (matched)
    * 	 server certificate expiration date OK
    * 	 server certificate activation date OK
    * 	 certificate public key: RSA
    * 	 certificate version: #3
    * 	 subject: CN=kiteci.com
    * 	 start date: Sun, 29 Oct 2017 21:07:37 GMT
    * 	 expire date: Sat, 27 Jan 2018 21:07:37 GMT
    * 	 issuer: C=US,O=Let's Encrypt,CN=Let's Encrypt Authority X3
    * 	 compression: NULL
    * ALPN, server accepted to use http/1.1
    > GET / HTTP/1.1
    > Host: kiteci.com
    > User-Agent: curl/7.47.0
    > Accept: */*
    >
    < HTTP/1.1 200 OK
    < Server: nginx/1.13.6
    < Date: Sun, 29 Oct 2017 22:31:59 GMT
    < Content-Type: text/html
    < Content-Length: 612
    < Last-Modified: Thu, 14 Sep 2017 16:35:09 GMT
    < ETag: "59baafbd-264"
    < Accept-Ranges: bytes
    < Strict-Transport-Security: max-age=15768000
    <
    <!DOCTYPE html>
    <html>
    <head>
    <title>Welcome to nginx!</title>
    <style>
        body {
            width: 35em;
            margin: 0 auto;
            font-family: Tahoma, Verdana, Arial, sans-serif;
        }
    </style>
    </head>
    <body>
    <h1>Welcome to nginx!</h1>
    <p>If you see this page, the nginx web server is successfully installed and
    working. Further configuration is required.</p>
    
    <p>For online documentation and support please refer to
    <a href="http://nginx.org/">nginx.org</a>.<br/>
    Commercial support is available at
    <a href="http://nginx.com/">nginx.com</a>.</p>
    
    <p><em>Thank you for using nginx.</em></p>
    </body>
    </html>
    * Connection #0 to host kiteci.com left intact
    ```
