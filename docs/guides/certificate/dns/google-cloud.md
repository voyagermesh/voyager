---
title: Issue Let's Encrypt certificate using Google Cloud DNS
description: Issue Let's Encrypt certificate using Google Cloud DNS in Kubernetes
menu:
  product_voyager_6.0.0:
    identifier: googlecloud-dns
    name: Google Cloud
    parent: dns-certificate
    weight: 15
product_name: voyager
menu_name: product_voyager_6.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Issue Let's Encrypt certificate using Google Cloud DNS

This tutorial shows how to issue free SSL certificate from Let's Encrypt via DNS challenge for domains using Google Cloud DNS service.

This article has been tested with a GKE cluster.

```console
$ kubectl version --short
Client Version: v1.8.5
Server Version: v1.8.5-gke.0
```

## Deploy Voyager operator

Deploy Voyager operator following instructions [here](/docs/setup/install.md).

```console
# install without RBAC
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0/hack/deploy/voyager.sh \
  | bash -s -- --provider=gke
```

If you are trying this on a RBAC enabled cluster, pass the flag `--rbac` to installer script.

```console
# install without RBAC
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0/hack/deploy/voyager.sh \
  | bash -s -- --provider=gke --rbac
```

## Setup Google Cloud DNS Zone

In this tutorial, I am going to use `kiteci.com` domain that was purchased on namecheap.com . Now, go to the [DNconsole.cloud.google.com/net-services/dns/zones) on your Google Cloud console and create a zone for this domain.

![create-zone](/docs/images/certificate/google-cloud/create-zone.png)

Once the zone is created, you can see the list of name servers in Google cloud console.

![ns-servers](/docs/images/certificate/google-cloud/ns-servers.png)

Now, go to the website of your domain registrar and update the list of name servers.

![domain-registrar](/docs/images/certificate/google-cloud/domain-registrar.png)

Give time to propagate the updated DNS records. You can use the following command to confirm that the name server records has been updated.

```console
$ dig -t ns kiteci.com

; <<>> DiG 9.10.3-P4-Ubuntu <<>> -t ns kiteci.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 60415
;; flags: qr rd ra; QUERY: 1, ANSWER: 4, AUTHORITY: 0, ADDITIONAL: 9

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 512
;; QUESTION SECTION:
;kiteci.com.			IN	NS

;; ANSWER SECTION:
kiteci.com.		21600	IN	NS	ns-cloud-e3.googledomains.com.
kiteci.com.		21600	IN	NS	ns-cloud-e4.googledomains.com.
kiteci.com.		21600	IN	NS	ns-cloud-e1.googledomains.com.
kiteci.com.		21600	IN	NS	ns-cloud-e2.googledomains.com.

;; ADDITIONAL SECTION:
ns-cloud-e1.googledomains.com. 143957 IN A	216.239.32.110
ns-cloud-e1.googledomains.com. 144007 IN AAAA	2001:4860:4802:32::6e
ns-cloud-e2.googledomains.com. 143976 IN A	216.239.34.110
ns-cloud-e2.googledomains.com. 144137 IN AAAA	2001:4860:4802:34::6e
ns-cloud-e3.googledomains.com. 144001 IN A	216.239.36.110
ns-cloud-e3.googledomains.com. 144532 IN AAAA	2001:4860:4802:36::6e
ns-cloud-e4.googledomains.com. 144141 IN A	216.239.38.110
ns-cloud-e4.googledomains.com. 144080 IN AAAA	2001:4860:4802:38::6e

;; Query time: 55 msec
;; SERVER: 127.0.1.1#53(127.0.1.1)
;; WHEN: Mon Dec 04 09:15:19 PST 2017
;; MSG SIZE  rcvd: 333
```

## Configure Service Account Permissions

To issue SSL certificate using Let's Encrypt, we have to prove that we own the `kiteci.com` domain. Voyager operator requires necessary permission to add and remove a TXT record for domain `_acme-challenge.<domain>` to complete the DNS challenge.

There are few different ways to grant these permissions to voyager operator pods.

### option 1: Create Service Account
If you are running cluster on cloud providers other than Google Cloud but want to use Google Cloud DNS as your DNS provider, this is your only option. You can also use this method for clusters running on Google Cloud.

Here we will create a new ServiceAccount called `voyager` in [Service Accounts console](https://console.cloud.google.com/iam-admin/serviceaccounts/project) and grant it `DNS Administrator` permission. Then we wil issue a json key for this service account and pass this to voyager using a Kubernetes secret.

![create-svc-account](/docs/images/certificate/google-cloud/create-svc-account.png)

```console
mv <your_service_account_key>.json GOOGLE_SERVICE_ACCOUNT_JSON_KEY

kubectl create secret generic voyager-gce --namespace default \
  --from-literal=GCE_PROJECT=INSERT_YOUR_PROJECT_ID_HERE \
  --from-file=GOOGLE_SERVICE_ACCOUNT_JSON_KEY

$ kubectl get secret voyager-gce -o yaml
apiVersion: v1
data:
  GCE_PROJECT: dGlnZXJ3b3Jrcy1rdWJl
  GOOGLE_SERVICE_ACCOUNT_JSON_KEY: ewogICJ0eXBlIj2VhY2NvdW50LmNvbSIKfQo=
kind: Secret
metadata:
  creationTimestamp: 2017-12-04T17:36:24Z
  name: voyager-gce
  namespace: default
  resourceVersion: "7372"
  selfLink: /api/v1/namespaces/default/secrets/voyager-gce
  uid: a612c439-d919-11e7-81d9-42010a8000db
type: Opaque
```

**NB**:

- The Kubernetes secret must be created in the same namespace where the `Certificate` object exists.

### option 2: Using Compute Engine Default Service Account
If your domains are hosted in the same Google Cloud project as your GKE cluster, you can use this mechanism. When you create your GKE cluster, enable `Cloud Platform` scope. This will allow voyager operator to update DNS records in this project.

![gke-permissions](/docs/images/certificate/google-cloud/gke-permissions.png)

**NB**:
- I don't know how to apply these permission for an existing GKE cluster. If you know how to do that, please send me to pr.

### option 3: Use `GOOGLE_APPLICATION_CREDENTIALS`
Voyager operator can load a json key file whose path is specified by the GOOGLE_APPLICATION_CREDENTIALS environment variable. To use this option, mount a json key file into voyager operator deployment.

## Create Certificate

Create a secret to provide ACME user email. Change the email to a valid email address and run the following command:

```console
kubectl create secret generic acme-account --from-literal=ACME_EMAIL=me@example.com
```

Create the Certificate CRD to issue TLS certificate from Let's Encrypt using DNS challenge.

```console
kubectl apply -f crt.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Certificate
metadata:
  name: kitecicom
  namespace: default
spec:
  domains:
  - kiteci.com
  - www.kiteci.com
  acmeUserSecretName: acme-account
  challengeProvider:
    dns:
      provider: gce
      credentialSecretName: voyager-gce
```

Now, voyager will perform domain validation by setting a TXT record for each domain by prepending the label `_acme-challenge`to the domain name being validated in this certificate. This TXT record will be removed after validation is complete. Once you successfully complete the challenges for a domain, the resulting authorization is cached for your account to use again later. Cached authorizations last for 30 days from the time of validation. If the certificate you requested has all of the necessary authorizations cached then validation will not happen again until the relevant cached authorizations expire.

![acme-challenge](/docs/images/certificate/google-cloud/acme-challenge.png)

After several minutes, you should see a new secret named `tls-kitecicom`. This contains the `tls.crt` and `tls.key` .

```console
$ kubectl get secrets
NAME                  TYPE                                  DATA      AGE
acme-account          Opaque                                3         12m
default-token-t4m4f   kubernetes.io/service-account-token   3         1h
tls-kitecicom         kubernetes.io/tls                     2         38s
voyager-gce           Opaque                                2         16m

$ kubectl describe secrets tls-kitecicom
Name:         tls-kitecicom
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  kubernetes.io/tls

Data
====
tls.crt:  3452 bytes
tls.key:  1675 bytes
```

```console
$ kubectl describe cert kitecicom
Name:         kitecicom
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"voyager.appscode.com/v1beta1","kind":"Certificate","metadata":{"annotations":{},"name":"kitecicom","namespace":"default"},"spec":{"acmeU...
API Version:  voyager.appscode.com/v1beta1
Kind:         Certificate
Metadata:
  Cluster Name:
  Creation Timestamp:             2017-12-04T17:50:07Z
  Deletion Grace Period Seconds:  <nil>
  Deletion Timestamp:             <nil>
  Generation:                     0
  Resource Version:               8514
  Self Link:                      /apis/voyager.appscode.com/v1beta1/namespaces/default/certificates/kitecicom
  UID:                            90e74603-d91b-11e7-81d9-42010a8000db
Spec:
  Acme User Secret Name:  acme-account
  Challenge Provider:
    Dns:
      Credential Secret Name:  voyager-gce
      Provider:                gce
  Domains:
    kiteci.com
    www.kiteci.com
Status:
  Conditions:
    Last Update Time:  2017-12-04T17:51:49Z
    Type:              Issued
  Last Issued Certificate:
    Account Ref:      https://acme-v01.api.letsencrypt.org/acme/reg/25335618
    Cert Stable URL:
    Cert URL:         https://acme-v01.api.letsencrypt.org/acme/cert/031f94c84a8b8634b3e58a8f2a9ac56013b8
    Not After:        2018-03-04T16:51:48Z
    Not Before:       2017-12-04T16:51:48Z
    Serial Number:    272083376884530266786654704451984654603192
Events:
  Type    Reason           Age   From              Message
  ----    ------           ----  ----              -------
  Normal  IssueSuccessful  1m    voyager-operator  Successfully issued certificate
  Normal  IssueSuccessful  1m    voyager operator  Successfully issued certificate
```

**NB**

- By default, voyager will store the issued SSL certificates in a secret named as `tls-<certificate-name>`. If you want to store the issued certificates in a different secret, you can provide that in that in the `spec.storage.secret.name` field in the `Certificate` object.

```console
$ cat crt-secret-store.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Certificate
metadata:
  name: kitecicom
  namespace: default
spec:
  domains:
  - kiteci.com
  - www.kiteci.com
  acmeUserSecretName: acme-account
  challengeProvider:
    dns:
      provider: gce
      credentialSecretName: voyager-gce
  storage:
    secret:
      name: cert-kitecipro
```

- If you enabled `Cloud Platform` scope for your GKE cluster (option 2), you don't need to set `spec.challengeProvider.dns.credentialSecretName` field.

```console
$ cat crt-gce.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Certificate
metadata:
  name: kitecicom
  namespace: default
spec:
  domains:
  - kiteci.com
  - www.kiteci.com
  acmeUserSecretName: acme-account
  challengeProvider:
    dns:
      provider: gce
```

## Configure Ingress

We are going to use two separate services as backend. Run the following commands to deploy backends:

```console
kubectl run nginx --image=nginx
kubectl expose deployment nginx --name=web --port=80 --target-port=80

kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4
kubectl expose deployment echoserver --name=echo --port=80 --target-port=8080
```

Now create Ingress `ing-tls.yaml`

```console
kubectl apply -f ing-tls.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/rewrite-target: /
spec:
  tls:
  - hosts:
    - www.kiteci.com
    ref:
      kind: Certificate
      name: kitecicom
  rules:
  - host: www.kiteci.com
    http:
      paths:
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
      - path: /
        backend:
          serviceName: echo
          servicePort: 80
```

Wait for the LoadBlanacer IP to be assigned. Once the IP is assigned, set the LoadBlancer IP as the A record for test domain `www.kiteci.com`

```console
$ kubectl get svc voyager-test-ingress -o wide
NAME                   TYPE           CLUSTER-IP     EXTERNAL-IP       PORT(S)                      AGE       SELECTOR
voyager-test-ingress   LoadBalancer   10.15.243.46   104.155.134.134   443:31886/TCP,80:31703/TCP   1m        origin-api-group=voyager.appscode.com,origin-name=test-ingress,origin=voyager
```

![a-record](/docs/images/certificate/google-cloud/a-record.png)

Now wait a bit for DNS to propagate. Run the following command to confirm DNS propagation.

```console
$ dig +short www.kiteci.com
10.15.243.46
```

Now open URL https://www.kiteci.com/web . This should show you the familiar nginx welcome page. If you visit https://www.kiteci.com , it will echo your connection info.
