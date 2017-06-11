node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    def INTERNAL_TAG
    def CLOUD_PROVIDER = "gce"
    def DEPLOYMENT_YAML
    def NODE
    def NAMESPACE

    stage("set env") {
        env.GOPATH = "${PWD}"
        env.GOBIN = "${GOPATH}/bin"
        env.PATH = "$env.PATH:${env.GOBIN}:/usr/local/go/bin"
    }
    try {
        dir("${project_dir}") {
            stage("checkout") {
                checkout scm
            }
            stage("builddeps") {
                sh "./hack/builddeps.sh"
            }
            stage("build binary") {
                sh "./hack/make.py"
            }
            stage("build docker") {
                sh "./hack/docker/voyager/setup.sh"
            }
            stage("detect tag") {
                INTERNAL_TAG = sh(
                        script: '. ./hack/libbuild/common/lib.sh > /dev/null && detect_tag > /dev/null && echo $TAG',
                        returnStdout: true
                ).trim()
            }
            stage("set namespace and deployment") {
                rand = sh(
                        script: 'cat /dev/urandom | tr -dc \'a-z0-9\' | fold -w 10 | head -n 1',
                        returnStdout: true
                ).trim()
                NAMESPACE =  "test-$rand"
                DEPLOYMENT_YAML = readFile('./hack/deploy/deployments.yaml').
                        replace('$CLOUD_PROVIDER', CLOUD_PROVIDER).
                        replace('$TAG', INTERNAL_TAG)
            }
            stage("docker push") {
                sh "docker push appscode/voyager:$INTERNAL_TAG"
            }
            stage("get node name") {
                NODE = sh(
                        script: "kubectl get nodes --selector=kubernetes.io/role=node -o jsonpath='{.items[0].metadata.name}'",
                        returnStdout: true
                ).trim()
            }
            stage("deploy in cluster") {
                sh "echo '$DEPLOYMENT_YAML' | kubectl create -f -"
            }
            stage("integration test") {
                sh "kubectl create namespace $NAMESPACE"
                sh "./hack/make.py test integration -cloud-provider=$CLOUD_PROVIDER -daemon-host-name=$NODE -namespace=$NAMESPACE -max-test=4"
            }
        }
        currentBuild.result = 'SUCCESS'
    } catch (Exception err) {
        println(err.getMessage())
        currentBuild.result = 'FAILURE'
    } finally {
        deleteDir()
        sh "kubectl delete deployments voyager-operator"
        sh "kubectl delete svc voyager-operator"
        sh "docker rmi -f appscode/voyager:$INTERNAL_TAG"
        if (NAMESPACE != null) {
            sh "kubectl delete namespace $NAMESPACE"
        }
    }
}
