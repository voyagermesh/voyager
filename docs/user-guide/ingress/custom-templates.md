# Using Custom Templates

```yaml
$ cat /tmp/defaults

defaults
	log global

	# https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.2-option%20abortonclose
	option dontlog-normal
	log /dev/log local0 notice alert
	option dontlognull
	option http-server-close

	# Timeout values
	timeout client 5s
	timeout client-fin 5s
	timeout connect 5s
	timeout server 5s
	timeout tunnel 5s

	# default traffic mode is http
	# mode is overwritten in case of tcp services
	mode http
```

kubectl create configmap -n kube-system voyager-templates --from-file=/tmp/defaults

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
        - --cloud-provider=minikube
        - --cloud-config=
        - --ingress-class=
        - '--custom-templates=/srv/voyager/custom/*'
        image: appscode/voyager:3.2.0-rc.2
        ports:
        - containerPort: 56790
          name: http
          protocol: TCP
        volumeMounts:
          - mountPath: /etc/kubernetes
            name: cloudconfig
            readOnly: true
          - mountPath: /srv/voyager/custom
            name: voyager-templates
            readOnly: true
      volumes:
        - hostPath:
            path: /etc/kubernetes
          name: cloudconfig
        - configMap:
          name: voyager-templates
EOF
```
