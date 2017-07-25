Create Deployment, secrets, service and ingress using kubectl.

```console
kubectl create -f deployments.yaml -f secrets.yaml -f svc.yaml -f ingress.yaml
```

When voyager-ssh-test service is created ad the CNAME/IP to domain.

ssh into first pod.
```console
ssh root@ssh.appscode.co
// Use password 1234
```

ssh into 2nd pod.
```console
ssh -p 8022 root@ssh-2.appscode.co
// Use password 1234
```
