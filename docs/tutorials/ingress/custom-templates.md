---
menu:
  product_voyager_5.0.0-rc.6:
    name: Custom Templates
    parent: ingress
    weight: 55
product_name: voyager
menu_name: product_voyager_5.0.0-rc.6
section_menu_id: tutorials
---


# Using Custom HAProxy Templates

Since 3.2.0, Voyager can use custom templates provided by users to render HAProxy configuration. Voyager comes with a set of GO [text/templates](https://golang.org/pkg/text/template/) found [here](/hack/docker/voyager/templates). These templates are mounted at `/srv/voyager/templates`. You can mount a ConfigMap with matching template names when installing Voyager operator to a different location and pass that to Voyager operator using `--custom-templates` flag. Voyager will [load](https://github.com/appscode/voyager/blob/3ae30cd023ff8fa6301d2656bf9fbc5765529691/pkg/haproxy/template.go#L40) the built-in templates first and then load any custom templates if provided. As long as the custom templates have [same name](https://golang.org/pkg/text/template/#Template.ParseGlob) as the built-in templates, custom templates will be used render HAProxy config. You can overwrite any number of templates as you wish. Also note that templates are loaded when Voyager operator starts. So, if you want to reload custom templates, you need to restart the running Voyager operator pod (not HAProxy pods).

In this example, we are going to overwrite the [defaults.cfg](/hack/docker/voyager/templates/defaults.cfg) template which is used to render the [`defaults`](https://github.com/appscode/voyager/blob/3ae30cd023ff8fa6301d2656bf9fbc5765529691/hack/docker/voyager/templates/haproxy.cfg#L6) section of HAProxy config.

```console
$ cat /tmp/defaults.cfg

defaults
	log global

	# my custom template
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

Now create a ConfigMap using the defaults.cfg as key and the file content as the value.
```console
$ kubectl create configmap -n kube-system voyager-templates --from-file=/tmp/defaults.cfg
```

Now, update the appropriate installer YAMLs in [`/hack/deploy/with*-rbac.yaml`](/hack/deploy) for your cluster to mount the ConfigMap `voyager-templates` and set `--custom-templates` flag. Below is the updated YAML for a minikube cluster without RBAC.

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
        - --custom-templates=/srv/voyager/custom/*.cfg
        image: appscode/voyager:5.0.0-rc.6
        ports:
        - containerPort: 56790
          name: http
          protocol: TCP
        volumeMounts:
          - mountPath: /etc/kubernetes
            name: cloudconfig
            readOnly: true
          - mountPath: /srv/voyager/custom
            name: templates
            readOnly: true
      volumes:
        - hostPath:
            path: /etc/kubernetes
          name: cloudconfig
        - configMap:
            name: voyager-templates
          name: templates
EOF
```

This will restart the Voyager operator pods. After start, Voyager pod will update any existing HAProxy configuration to the new templates.
