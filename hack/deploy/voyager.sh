#!/bin/bash

# Copyright The Voyager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eou pipefail

crds=(certificates ingresses)

echo "checking kubeconfig context"
kubectl config current-context || {
    echo "Set a context (kubectl use-context <context>) out of the following:"
    echo
    kubectl config get-contexts
    exit 1
}
echo ""

OS=""
ARCH=""
DOWNLOAD_URL=""
DOWNLOAD_DIR=""
TEMP_DIRS=()
ONESSL=""
ONESSL_VERSION=v0.13.1

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup() {
    rm -rf ca.crt ca.key server.crt server.key
    # remove temporary directories
    for dir in "${TEMP_DIRS[@]}"; do
        rm -rf "${dir}"
    done
}

# detect operating system
# ref: https://raw.githubusercontent.com/helm/helm/master/scripts/get
function detectOS() {
    OS=$(echo $(uname) | tr '[:upper:]' '[:lower:]')

    case "$OS" in
        # Minimalist GNU for Windows
        cygwin* | mingw* | msys*) OS='windows' ;;
    esac
}

# detect machine architecture
function detectArch() {
    ARCH=$(uname -m)
    case $ARCH in
        armv7*) ARCH="arm" ;;
        aarch64) ARCH="arm64" ;;
        x86) ARCH="386" ;;
        x86_64) ARCH="amd64" ;;
        i686) ARCH="386" ;;
        i386) ARCH="386" ;;
    esac
}

detectOS
detectArch

# download file pointed by DOWNLOAD_URL variable
# store download file to the directory pointed by DOWNLOAD_DIR variable
# you have to sent the output file name as argument. i.e. downloadFile myfile.tar.gz
function downloadFile() {
    if curl --output /dev/null --silent --head --fail "$DOWNLOAD_URL"; then
        curl -fsSL ${DOWNLOAD_URL} -o $DOWNLOAD_DIR/$1
    else
        echo "File does not exist"
        exit 1
    fi
}

export APPSCODE_ENV=${APPSCODE_ENV:-prod}
trap cleanup EXIT

# ref: https://github.com/appscodelabs/libbuild/blob/master/common/lib.sh#L55
inside_git_repo() {
    git rev-parse --is-inside-work-tree >/dev/null 2>&1
    inside_git=$?
    if [ "$inside_git" -ne 0 ]; then
        echo "Not inside a git repository"
        exit 1
    fi
}

detect_tag() {
    inside_git_repo

    # http://stackoverflow.com/a/1404862/3476121
    git_tag=$(git describe --exact-match --abbrev=0 2>/dev/null || echo '')

    commit_hash=$(git rev-parse --verify HEAD)
    git_branch=$(git rev-parse --abbrev-ref HEAD)
    commit_timestamp=$(git show -s --format=%ct)

    if [ "$git_tag" != '' ]; then
        TAG=$git_tag
        TAG_STRATEGY='git_tag'
    elif [ "$git_branch" != 'master' ] && [ "$git_branch" != 'HEAD' ] && [[ "$git_branch" != release-* ]]; then
        TAG=$git_branch
        TAG_STRATEGY='git_branch'
    else
        hash_ver=$(git describe --tags --always --dirty)
        TAG="${hash_ver}"
        TAG_STRATEGY='commit_hash'
    fi

    export TAG
    export TAG_STRATEGY
    export git_tag
    export git_branch
    export commit_hash
    export commit_timestamp
}

onessl_found() {
    # https://stackoverflow.com/a/677212/244009
    if [ -x "$(command -v onessl)" ]; then
        onessl version --check=">=${ONESSL_VERSION}" >/dev/null 2>&1 || {
            # old version of onessl found
            echo "Found outdated onessl"
            return 1
        }
        export ONESSL=onessl
        return 0
    fi
    return 1
}

# download onessl if it does not exist
onessl_found || {
    echo "Downloading onessl ..."

    ARTIFACT="https://github.com/kubepack/onessl/releases/download/${ONESSL_VERSION}"
    ONESSL_BIN=onessl-${OS}-${ARCH}
    case "$OS" in
        cygwin* | mingw* | msys*)
            ONESSL_BIN=${ONESSL_BIN}.exe
            ;;
    esac

    DOWNLOAD_URL=${ARTIFACT}/${ONESSL_BIN}
    DOWNLOAD_DIR="$(mktemp -dt onessl-XXXXXX)"
    TEMP_DIRS+=($DOWNLOAD_DIR) # store DOWNLOAD_DIR to cleanup later

    downloadFile $ONESSL_BIN # downloaded file name will be saved as the value of ONESSL_BIN variable

    export ONESSL=${DOWNLOAD_DIR}/${ONESSL_BIN}
    chmod +x $ONESSL
}

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export VOYAGER_NAMESPACE=kube-system
export VOYAGER_SERVICE_ACCOUNT=voyager-operator
export VOYAGER_RUN_ON_MASTER=0
export VOYAGER_ENABLE_VALIDATING_WEBHOOK=false
export VOYAGER_RESTRICT_TO_NAMESPACE=false
export VOYAGER_ROLE_TYPE=ClusterRole
export VOYAGER_DOCKER_REGISTRY=${DOCKER_REGISTRY:-appscode}
export VOYAGER_IMAGE_TAG=${VOYAGER_IMAGE_TAG:-v11.0.1}
export VOYAGER_HAPROXY_IMAGE_TAG=1.9.6-v11.0.1-alpine
export VOYAGER_IMAGE_PULL_SECRET=
export VOYAGER_IMAGE_PULL_POLICY=IfNotPresent
export VOYAGER_ENABLE_ANALYTICS=true
export VOYAGER_UNINSTALL=0
export VOYAGER_PURGE=0
export VOYAGER_TEMPLATE_CONFIGMAP=
export VOYAGER_ENABLE_STATUS_SUBRESOURCE=false
export VOYAGER_BYPASS_VALIDATING_WEBHOOK_XRAY=false
export VOYAGER_USE_KUBEAPISERVER_FQDN_FOR_AKS=true
export VOYAGER_PRIORITY_CLASS=system-cluster-critical
export VOYAGER_INGRESS_CLASS_OVERRIDE=

export SCRIPT_LOCATION="curl -fsSL https://raw.githubusercontent.com/appscode/voyager/v11.0.1/"
if [[ "$APPSCODE_ENV" == "dev" ]]; then
    detect_tag
    export SCRIPT_LOCATION="cat "
    export VOYAGER_IMAGE_TAG=$TAG
    export VOYAGER_HAPROXY_IMAGE_TAG=1.9.6-$TAG-alpine
    export VOYAGER_IMAGE_PULL_POLICY=Always
fi

KUBE_APISERVER_VERSION=$(kubectl version -o=json | $ONESSL jsonpath '{.serverVersion.gitVersion}')
$ONESSL semver --check='<1.9.0' $KUBE_APISERVER_VERSION || { export VOYAGER_ENABLE_VALIDATING_WEBHOOK=true; }
$ONESSL semver --check='<1.11.0' $KUBE_APISERVER_VERSION || { export VOYAGER_ENABLE_STATUS_SUBRESOURCE=true; }

export VOYAGER_WEBHOOK_SIDE_EFFECTS=
$ONESSL semver --check='<1.12.0' $KUBE_APISERVER_VERSION || { export VOYAGER_WEBHOOK_SIDE_EFFECTS='sideEffects: None'; }

show_help() {
    echo "voyager.sh - install voyager operator"
    echo " "
    echo "voyager.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                             show brief help"
    echo "-n, --namespace=NAMESPACE              specify namespace (default: kube-system)"
    echo "-p, --provider=PROVIDER                specify a cloud provider"
    echo "    --ingress-class=CLASS              specify an ingress class"
    echo "    --docker-registry                  docker registry used to pull voyager images (default: appscode)"
    echo "    --haproxy-image-tag                tag of Docker image containing HAProxy binary (default: 1.9.6-v11.0.1-alpine)"
    echo "    --image-pull-secret                name of secret used to pull voyager operator images"
    echo "    --restrict-to-namespace            restrict voyager to its own namespace"
    echo "    --run-on-master                    run voyager operator on master"
    echo "    --enable-validating-webhook        enable/disable validating webhooks for voyager CRDs"
    echo "    --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks"
    echo "    --template-cfgmap=CONFIGMAP        name of configmap with custom templates"
    echo "    --enable-status-subresource        if enabled, uses status sub resource for Voyager crds"
    echo "    --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)"
    echo "    --enable-analytics                 send usage events to Google Analytics (default: true)"
    echo "    --uninstall                        uninstall voyager"
    echo "    --purge                            purges Voyager crd objects and crds"
}

while test $# -gt 0; do
    case "$1" in
        -h | --help)
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
            export VOYAGER_NAMESPACE=$(echo $1 | sed -e 's/^[^=]*=//g')
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
            export VOYAGER_CLOUD_PROVIDER=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --ingress-class*)
            export VOYAGER_INGRESS_CLASS_OVERRIDE="$(echo $1 | sed -e 's/^[^=]*=//g')"
            shift
            ;;
        --docker-registry*)
            export VOYAGER_DOCKER_REGISTRY=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --haproxy-image-tag*)
            export VOYAGER_HAPROXY_IMAGE_TAG=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --image-pull-secret*)
            secret=$(echo $1 | sed -e 's/^[^=]*=//g')
            export VOYAGER_IMAGE_PULL_SECRET="name: '$secret'"
            shift
            ;;
        --enable-validating-webhook*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export VOYAGER_ENABLE_VALIDATING_WEBHOOK=false
            else
                export VOYAGER_ENABLE_VALIDATING_WEBHOOK=true
            fi
            shift
            ;;
        --bypass-validating-webhook-xray*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export VOYAGER_BYPASS_VALIDATING_WEBHOOK_XRAY=false
            else
                export VOYAGER_BYPASS_VALIDATING_WEBHOOK_XRAY=true
            fi
            shift
            ;;
        --enable-status-subresource*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export VOYAGER_ENABLE_STATUS_SUBRESOURCE=false
            fi
            shift
            ;;
        --use-kubeapiserver-fqdn-for-aks*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export VOYAGER_USE_KUBEAPISERVER_FQDN_FOR_AKS=false
            else
                export VOYAGER_USE_KUBEAPISERVER_FQDN_FOR_AKS=true
            fi
            shift
            ;;
        --enable-analytics*)
            val=$(echo $1 | sed -e 's/^[^=]*=//g')
            if [ "$val" = "false" ]; then
                export VOYAGER_ENABLE_ANALYTICS=false
            fi
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
            export VOYAGER_TEMPLATE_CONFIGMAP=$(echo $1 | sed -e 's/^[^=]*=//g')
            shift
            ;;
        --uninstall)
            export VOYAGER_UNINSTALL=1
            shift
            ;;
        --purge)
            export VOYAGER_PURGE=1
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

if [ "$VOYAGER_NAMESPACE" != "kube-system" ]; then
    export VOYAGER_PRIORITY_CLASS=""
fi

if [ "$VOYAGER_UNINSTALL" -eq 1 ]; then
    # delete webhooks and apiservices
    kubectl delete validatingwebhookconfiguration -l app=voyager || true
    kubectl delete mutatingwebhookconfiguration -l app=voyager || true
    kubectl delete apiservice -l app=voyager
    # delete voyager operator
    kubectl delete deployment -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete service -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete secret -l app=voyager --namespace $VOYAGER_NAMESPACE
    # delete RBAC objects
    kubectl delete serviceaccount -l app=voyager --namespace $VOYAGER_NAMESPACE
    # skip deleting clusterrole & clusterrolebinding in case used by --restrict-to-namespace mode
    # kubectl delete clusterrolebindings -l app=voyager
    # kubectl delete clusterrole -l app=voyager
    kubectl delete rolebindings -l app=voyager --namespace $VOYAGER_NAMESPACE
    kubectl delete role -l app=voyager --namespace $VOYAGER_NAMESPACE

    echo "waiting for voyager operator pod to stop running"
    for (( ; ; )); do
        pods=($(kubectl get pods -n $VOYAGER_NAMESPACE -l app=voyager -o jsonpath='{range .items[*]}{.metadata.name} {end}'))
        total=${#pods[*]}
        if [ $total -eq 0 ]; then
            break
        fi
        sleep 2
    done

    # https://github.com/kubernetes/kubernetes/issues/60538
    if [ "$VOYAGER_PURGE" -eq 1 ]; then
        for crd in "${crds[@]}"; do
            pairs=($(kubectl get ${crd}.voyager.appscode.com --all-namespaces -o jsonpath='{range .items[*]}{.metadata.name} {.metadata.namespace} {end}' || true))
            total=${#pairs[*]}

            # save objects
            if [ $total -gt 0 ]; then
                echo "dumping ${crd} objects into ${crd}.yaml"
                kubectl get ${crd}.voyager.appscode.com --all-namespaces -o yaml >${crd}.yaml
            fi

            for ((i = 0; i < $total; i += 2)); do
                name=${pairs[$i]}
                namespace=${pairs[$i + 1]}
                # remove finalizers
                kubectl patch ${crd}.voyager.appscode.com $name -n $namespace -p '{"metadata":{"finalizers":[]}}' --type=merge
                # delete crd object
                echo "deleting ${crd} $namespace/$name"
                kubectl delete ${crd}.voyager.appscode.com $name -n $namespace
            done

            # delete crd
            kubectl delete crd ${crd}.voyager.appscode.com || true
        done
        # delete user roles
        kubectl delete clusterroles appscode:voyager:edit appscode:voyager:view
    fi

    echo
    echo "Successfully uninstalled Voyager!"
    exit 0
fi

echo "checking whether extended apiserver feature is enabled"
$ONESSL has-keys configmap --namespace=kube-system --keys=requestheader-client-ca-file extension-apiserver-authentication || {
    echo "Set --requestheader-client-ca-file flag on Kubernetes apiserver"
    exit 1
}
echo ""

export KUBE_CA=
if [ "$VOYAGER_ENABLE_VALIDATING_WEBHOOK" = true ]; then
    $ONESSL get kube-ca >/dev/null 2>&1 || {
        echo "Admission webhooks can't be used when kube apiserver is accesible without verifying its TLS certificate (insecure-skip-tls-verify : true)."
        echo
        exit 1
    }
    export KUBE_CA=$($ONESSL get kube-ca | $ONESSL base64)
fi

case "$VOYAGER_CLOUD_PROVIDER" in
    aws)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
        ;;
    acs)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
        ;;
    aks)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
        ;;
    azure)
        export VOYAGER_CLOUD_CONFIG=/etc/kubernetes/azure.json
        export VOYAGER_INGRESS_CLASS=
        ;;
    baremetal)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
        ;;
    digitalocean)
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
    linode)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
        ;;
    metallb)
        export VOYAGER_CLOUD_CONFIG=
        export VOYAGER_INGRESS_CLASS=
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

if [ -n "$VOYAGER_INGRESS_CLASS_OVERRIDE" ]; then
    export VOYAGER_INGRESS_CLASS="$VOYAGER_INGRESS_CLASS_OVERRIDE"
fi

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

${SCRIPT_LOCATION}hack/deploy/operator.yaml | $ONESSL envsubst | kubectl apply -f -

if [ -n "$VOYAGER_TEMPLATE_CONFIGMAP" ]; then
    kubectl get configmap -n $VOYAGER_NAMESPACE $VOYAGER_TEMPLATE_CONFIGMAP >/dev/null 2>&1
    if [ "$?" -ne 0 ]; then
        echo "Missing configmap $VOYAGER_NAMESPACE/$VOYAGER_TEMPLATE_CONFIGMAP"
        exit 1
    fi
    kubectl patch deploy voyager-operator -n $VOYAGER_NAMESPACE \
        --patch="$(${SCRIPT_LOCATION}hack/deploy/use-custom-tpl.yaml | $ONESSL envsubst)"
fi

${SCRIPT_LOCATION}hack/deploy/service-account.yaml | $ONESSL envsubst | kubectl apply -f -
${SCRIPT_LOCATION}hack/deploy/rbac-list.yaml | $ONESSL envsubst | kubectl auth reconcile -f -
${SCRIPT_LOCATION}hack/deploy/user-roles.yaml | $ONESSL envsubst | kubectl auth reconcile -f -

if [ "$VOYAGER_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy voyager-operator -n $VOYAGER_NAMESPACE \
        --patch="$(${SCRIPT_LOCATION}hack/deploy/run-on-master.yaml)"
fi

if [ "$VOYAGER_ENABLE_VALIDATING_WEBHOOK" = true ]; then
    ${SCRIPT_LOCATION}hack/deploy/apiservices.yaml | $ONESSL envsubst | kubectl apply -f -
    ${SCRIPT_LOCATION}hack/deploy/validating-webhook.yaml | $ONESSL envsubst | kubectl apply -f -
fi

echo
echo "waiting until voyager operator deployment is ready"
$ONESSL wait-until-ready deployment voyager-operator --namespace $VOYAGER_NAMESPACE || {
    echo "Voyager operator deployment failed to be ready"
    exit 1
}

if [ "$VOYAGER_ENABLE_VALIDATING_WEBHOOK" = true ]; then
    echo "waiting until voyager apiservice is available"
    $ONESSL wait-until-ready apiservice v1beta1.admission.voyager.appscode.com || {
        echo "Voyager apiservice failed to be ready"
        exit 1
    }
fi

echo "waiting until voyager crds are ready"
for crd in "${crds[@]}"; do
    $ONESSL wait-until-ready crd ${crd}.voyager.appscode.com || {
        echo "$crd crd failed to be ready"
        exit 1
    }
done

if [ "$VOYAGER_ENABLE_VALIDATING_WEBHOOK" = true ]; then
    echo "checking whether admission webhook(s) are activated or not"
    active=$($ONESSL wait-until-has annotation \
        --apiVersion=apiregistration.k8s.io/v1beta1 \
        --kind=APIService \
        --name=v1beta1.admission.voyager.appscode.com \
        --key=admission-webhook.appscode.com/active \
        --timeout=5m || {
        echo
        echo "Failed to check if admission webhook(s) are activated or not. Please check operator logs to debug further."
        exit 1
    })
    if [ "$active" = false ]; then
        echo
        echo "Admission webhooks are not activated."
        echo "Enable it by configuring --enable-admission-plugins flag of kube-apiserver."
        echo "For details, visit: https://appsco.de/kube-apiserver-webhooks ."
        echo "After admission webhooks are activated, please uninstall and then reinstall Voyager operator."
        # uninstall misconfigured webhooks to avoid failures
        kubectl delete validatingwebhookconfiguration -l app=voyager || true
        exit 1
    fi
fi

echo
echo "Successfully installed Voyager in $VOYAGER_NAMESPACE namespace!"
