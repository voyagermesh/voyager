#!/bin/bash

echo "checking kubeconfig context"
kubectl config current-context || { echo "Set a context (kubectl use-context <context>) out of the following:"; echo; kubectl config get-contexts; exit 1; }
echo ""

# ref: https://stackoverflow.com/a/27776822/244009
case "$(uname -s)" in
    Darwin)
        curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-darwin-amd64
        chmod +x onessl
        export ONESSL=./onessl
        ;;

    Linux)
        curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-linux-amd64
        chmod +x onessl
        export ONESSL=./onessl
        ;;

    CYGWIN*|MINGW32*|MSYS*)
        curl -fsSL -o onessl.exe https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-windows-amd64.exe
        chmod +x onessl.exe
        export ONESSL=./onessl.exe
        ;;
    *)
        echo 'other OS'
        ;;
esac

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup {
    rm -rf $ONESSL ca.crt ca.key server.crt server.key
}
trap cleanup EXIT

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export VOYAGER_NAMESPACE=kube-system
export VOYAGER_SERVICE_ACCOUNT=default
export VOYAGER_ENABLE_RBAC=false
export VOYAGER_RUN_ON_MASTER=0
export VOYAGER_RESTRICT_TO_NAMESPACE=false
export VOYAGER_ROLE_TYPE=ClusterRole
export VOYAGER_DOCKER_REGISTRY=appscode
export VOYAGER_IMAGE_PULL_SECRET=
export VOYAGER_UNINSTALL=0

KUBE_APISERVER_VERSION=$(kubectl version -o=json | $ONESSL jsonpath '{.serverVersion.gitVersion}')
$ONESSL semver --check='>=1.9.0' $KUBE_APISERVER_VERSION
if [ $? -eq 0 ]; then
    export VOYAGER_ENABLE_ADMISSION_WEBHOOK=true
fi

show_help() {
    echo "voyager.sh - install voyager operator"
    echo " "
    echo "voyager.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "-n, --namespace=NAMESPACE          specify namespace (default: kube-system)"
    echo "-p, --provider=PROVIDER            specify a cloud provider"
    echo "    --rbac                         create RBAC roles and bindings"
    echo "    --docker-registry              docker registry used to pull voyager images (default: appscode)"
    echo "    --image-pull-secret            name of secret used to pull voyager operator images"
    echo "    --restrict-to-namespace        restrict voyager to its own namespace"
    echo "    --run-on-master                run voyager operator on master"
    echo "    --enable-admission-webhook     configure admission webhook for voyager CRDs"
    echo "    --template-cfgmap=CONFIGMAP    name of configmap with custom templates"
    echo "    --uninstall                    uninstall voyager"
}

while test $# -gt 0; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        -n)
            shift
            if test $# -gt 0; then
                export VOYAGER_NAMESPACE=$1
            else
                echo "no namespace specified"
                exit 1
            fi
            shift
            ;;
        --namespace*)
            export VOYAGER_NAMESPACE=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        -p)
            shift
            if test $# -gt 0; then
                export VOYAGER_CLOUD_PROVIDER=$1
            else
                echo "no provider specified"
                exit 1
            fi
            shift
            ;;
        --provider*)
            export VOYAGER_CLOUD_PROVIDER=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --docker-registry*)
            export VOYAGER_DOCKER_REGISTRY=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --image-pull-secret*)
            secret=`echo $1 | sed -e 's/^[^=]*=//g'`
            export VOYAGER_IMAGE_PULL_SECRET="name: '$secret'"
            shift
            ;;
        --enable-admission-webhook*)
            val=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ "$val" = "false" ]; then
                export VOYAGER_ENABLE_ADMISSION_WEBHOOK=false
            else
                export VOYAGER_ENABLE_ADMISSION_WEBHOOK=true
            fi
            shift
            ;;
        --rbac)
            export VOYAGER_SERVICE_ACCOUNT=voyager-operator
            export VOYAGER_ENABLE_RBAC=true
            shift
            ;;
        --run-on-master)
            export VOYAGER_RUN_ON_MASTER=1
            shift
            ;;
        --restrict-to-namespace)
            export VOYAGER_RESTRICT_TO_NAMESPACE=true
            export VOYAGER_ROLE_TYPE=Role
            shift
            ;;
        --template-cfgmap*)
            export VOYAGER_TEMPLATE_CONFIGMAP=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --uninstall)
            export VOYAGER_UNINSTALL=1
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

if [ "$VOYAGER_UNINSTALL" -eq 1 ]; then
    kubectl delete deployment -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete service -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete secret -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete apiservice -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete validatingwebhookconfiguration -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete mutatingwebhookconfiguration -l app=voyager --namespace $VOYAGER_NAMESPACE
    # Delete RBAC objects, if --rbac flag was used.
    kubectl delete serviceaccount -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete clusterrolebindings -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete clusterrole -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete rolebindings -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete role -l app=voyager --namespace $VOYAGER_NAMESPACE

    exit 0
fi

case "$VOYAGER_CLOUD_PROVIDER" in
	acs)
		export VOYAGER_CLOUD_CONFIG=/etc/kubernetes/azure.json
		export VOYAGER_INGRESS_CLASS=
		;;
	aws)
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=
		;;
	azure)
		export VOYAGER_CLOUD_CONFIG=/etc/kubernetes/azure.json
		export VOYAGER_INGRESS_CLASS=
		;;
	baremetal)
		export VOYAGER_CLOUD_PROVIDER=
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=
		;;
	gce)
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=
		;;
	gke)
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=voyager
		if [ "$VOYAGER_RUN_ON_MASTER" -eq 1 ]; then
			echo "GKE clusters do not provide access to master instance(s). Ignoring --run-on-master flag."
			export VOYAGER_RUN_ON_MASTER=0
		fi
		;;
	minikube)
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=
		;;
	openstack)
		export VOYAGER_CLOUD_CONFIG=
		export VOYAGER_INGRESS_CLASS=
		;;
	*)
		echo "Unknown provider = $VOYAGER_CLOUD_PROVIDER"
		show_help
		exit 1
		;;
esac

env | sort | grep VOYAGER*
echo ""


# create necessary TLS certificates:
# - a local CA key and cert
# - a webhook server key and cert signed by the local CA
$ONESSL create ca-cert
$ONESSL create server-cert server --domains=voyager-operator.$VOYAGER_NAMESPACE.svc
export SERVICE_SERVING_CERT_CA=$(cat ca.crt | $ONESSL base64)
export TLS_SERVING_CERT=$(cat server.crt | $ONESSL base64)
export TLS_SERVING_KEY=$(cat server.key | $ONESSL base64)
export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)

curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/operator.yaml | $ONESSL envsubst | kubectl apply -f -

if [ -n "$VOYAGER_TEMPLATE_CONFIGMAP" ]; then
	kubectl get configmap -n $VOYAGER_NAMESPACE $VOYAGER_TEMPLATE_CONFIGMAP >/dev/null 2>&1
	if [ "$?" -ne 0 ]; then
		echo "Missing configmap $VOYAGER_NAMESPACE/$VOYAGER_TEMPLATE_CONFIGMAP"
		exit 1
	fi
    kubectl patch deploy voyager-operator -n $VOYAGER_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/use-custom-tpl.yaml | $ONESSL envsubst)"
fi

if [ "$VOYAGER_ENABLE_RBAC" = true ]; then
    kubectl create serviceaccount $VOYAGER_SERVICE_ACCOUNT --namespace $VOYAGER_NAMESPACE
    kubectl label serviceaccount $VOYAGER_SERVICE_ACCOUNT app=voyager --namespace $VOYAGER_NAMESPACE
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/rbac-list.yaml | $ONESSL envsubst | kubectl auth reconcile -f -
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/user-roles.yaml | $ONESSL envsubst | kubectl auth reconcile -f -
fi

if [ "$VOYAGER_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy voyager-operator -n $VOYAGER_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/run-on-master.yaml)"
fi

if [ "$VOYAGER_ENABLE_ADMISSION_WEBHOOK" = true ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-rc.1/hack/deploy/admission.yaml | $ONESSL envsubst | kubectl apply -f -
fi
