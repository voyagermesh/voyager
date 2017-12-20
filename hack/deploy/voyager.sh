#!/bin/bash

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export VOYAGER_NAMESPACE=kube-system
export VOYAGER_SERVICE_ACCOUNT=default
export VOYAGER_ENABLE_RBAC=false
export VOYAGER_RUN_ON_MASTER=0
export VOYAGER_RESTRICT_TO_NAMESPACE=false
export VOYAGER_ROLE_TYPE=ClusterRole

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
    echo "    --restrict-to-namespace        restrict voyager to its own namespace"
    echo "    --run-on-master                run voyager operator on master"
    echo "    --template-cfgmap=CONFIGMAP    name of configmap with custom templates"
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
        *)
            show_help
            exit 1
            ;;
    esac
done

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

if [ -z "$VOYAGER_TEMPLATE_CONFIGMAP" ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.8/hack/deploy/operator.yaml | envsubst | kubectl apply -f -
else
	kubectl get configmap -n $VOYAGER_NAMESPACE $VOYAGER_TEMPLATE_CONFIGMAP >/dev/null 2>&1
	if [ "$?" -ne 0 ]; then
		echo "Missing configmap $VOYAGER_NAMESPACE/$VOYAGER_TEMPLATE_CONFIGMAP"
		exit 1
	fi
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.8/hack/deploy/operator-with-custom-tpl.yaml | envsubst | kubectl apply -f -
fi

if [ "$VOYAGER_ENABLE_RBAC" = true ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.8/hack/deploy/rbac.yaml | envsubst | kubectl apply -f -
fi

if [ "$VOYAGER_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy voyager-operator -n $VOYAGER_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.8/hack/deploy/run-on-master.yaml)"
fi
