```console
cat > /tmp/defaults <<EOF
EOF
```

kubectl create configmap -n kube-system voyager-template --from-file=/tmp/defaults

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: voyager
  name: voyager-operator
  namespace: kube-system
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: voyager
    spec:
      containers:
      - name: voyager
        args:
        - run
        - --v=3
        - --cloud-provider=$CLOUD_PROVIDER
        - --cloud-config=$CLOUD_CONFIG # ie. /etc/kubernetes/azure.json for azure
        - --ingress-class=$INGRESS_CLASS
        image: appscode/voyager:3.2.0-rc.2
        ports:
        - containerPort: 56790
          name: http
          protocol: TCP
        volumeMounts:
          - mountPath: /etc/kubernetes
            name: cloudconfig
            readOnly: true
      volumes:
        - hostPath:
            path: /etc/kubernetes
          name: cloudconfig
EOF
```