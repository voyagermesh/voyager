node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    def IMAGE = "appscode/voyager"
    def INTERNAL_TAG
    def CLOUD_PROVIDER = "gce"
    def CLUSTER_NAME = "clusterc"

    stage("set env") {
        env.GOPATH = "${PWD}"
        env.GOBIN = "${GOPATH}/bin"
        env.PATH = "$env.PATH:${env.GOBIN}:/usr/local/go/bin"
        sh "mkdir -p ${env.GOBIN}"
    }
    try {
        stage('builddeps') {
            sh 'sudo apt update &&\
        sudo apt install -y software-properties-common python-software-properties python-dev libyaml-dev python-pip build-essential curl &&\
        sudo pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage &&\
        go get -u golang.org/x/tools/cmd/goimports'
//      go get -u github.com/sgotti/glide-vc &&\
//      curl https://glide.sh/get | sh'
        }
        dir("${project_dir}") {
            stage("checkout") {
                checkout scm
            }
            stage("builddeps") {
                sh "sudo ./hack/builddeps.sh"
            }
//          stage("dependency") {
//              sh "glide slow"
//          }
            stage("build binary") {
                sh "./hack/make.py"
            }
            stage("build docker") {
                sh "./hack/docker/voyager/setup.sh"
            }
            stage("detect tag") {
                INTERNAL_TAG = sh(
                        script: '. ./hack/libbuild/common/lib.sh && detect_tag > /dev/null && echo $TAG',
                        returnStdout: true
                ).trim()
            }
            stage("docker push") {
                sh "docker push $IMAGE:$INTERNAL_TAG"
            }
            stage("deploy in cluster") {
                sh "kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/ingress.yaml"
                sh "kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/certificate.yaml"
                deployment_yaml = "apiVersion: extensions/v1beta1\n" +
                        "kind: Deployment\n" +
                        "metadata:\n" +
                        "  labels:\n" +
                        "    run: voyager-operator\n" +
                        "  name: voyager-operator\n" +
                        "  namespace: default\n" +
                        "spec:\n" +
                        "  replicas: 1\n" +
                        "  selector:\n" +
                        "    matchLabels:\n" +
                        "      run: voyager-operator\n" +
                        "  template:\n" +
                        "    metadata:\n" +
                        "      labels:\n" +
                        "        run: voyager-operator\n" +
                        "    spec:\n" +
                        "      containers:\n" +
                        "      - name: voyager-operator\n" +
                        "        args:\n" +
                        "        - --cloud-provider=${CLOUD_PROVIDER}\n" +
                        "        - --cluster-name=${CLUSTER_NAME}\n" +
                        "        - --v=3\n" +
                        "        image: ${IMAGE}:${INTERNAL_TAG}\n" +
                        "        ports:\n" +
                        "        - containerPort: 1234\n" +
                        "          name: zero\n" +
                        "          protocol: TCP\n" +
                        "---\n" +
                        "apiVersion: v1\n" +
                        "kind: Service\n" +
                        "metadata:\n" +
                        "  labels:\n" +
                        "    run: voyager-operator\n" +
                        "  name: voyager-operator\n" +
                        "spec:\n" +
                        "  ports:\n" +
                        "  - name: zero\n" +
                        "    port: 1234\n" +
                        "    targetPort: zero\n" +
                        "  selector:\n" +
                        "    run: voyager-operator"
                sh "echo '$deployment_yaml' | kubectl create -f -"
            }
            stage("integration test") {
                sh "./hack/make.py test integration -cloud-provider=$CLOUD_PROVIDER -cluster-name=$CLUSTER_NAME"
            }
        }
        currentBuild.result = 'SUCCESS'
    } catch (Exception err) {
        currentBuild.result = 'FAILURE'
    } finally {
        deleteDir()
        sh "docker rmi -f $IMAGE:$INTERNAL_TAG"
    }
}
