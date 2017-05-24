node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    def IMAGE = "appscode/voyager"
    def INTERNAL_TAG
    def cloud_provier = "gce"
    def cluster_name = "clusterc"
    def deployment_yaml
    def deployment_tmpl = '''
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    run: voyager-operator
  name: voyager-operator
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      run: voyager-operator
  template:
    metadata:
      labels:
        run: voyager-operator
    spec:
      containers:
      - name: voyager-operator
        args:
        - --cloud-provider=${CLOUD_PROVIDER}
        - --cluster-name=${CLUSTER_NAME}
        - --v=3
        image: ${IMAGE}
        ports:
        - containerPort: 1234
          name: zero
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  labels:
    run: voyager-operator
  name: voyager-operator
spec:
  ports:
  - name: zero
    port: 1234
    targetPort: zero
  selector:
    run: voyager-operator'''
    def template = new groovy.text.StreamingTemplateEngine().createTemplate(deployment_tmpl)

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
        go get -u golang.org/x/tools/cmd/goimports &&\
        go get -u github.com/sgotti/glide-vc &&\
        curl https://glide.sh/get | sh'
        }
        dir("${project_dir}") {
            stage("checkout") {
                checkout scm
            }
            stage("builddeps") {
                sh "sudo ./hack/builddeps.sh"
            }
            stage("dependency") {
                sh "glide slow"
            }
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
                def binding = [
                        CLOUD_PROVIDER: "$cloud_provier",
                        CLUSTER_NAME: "$cluster_name",
                        IMAGE: "$IMAGE:$INTERNAL_TAG"
                ]
                deployment_yaml = template.make(binding)
                println(deployment_yaml)
            }
            stage("docker push") {
                sh "docker push $IMAGE:$INTERNAL_TAG"
            }
            stage("deploy in cluster") {
                sh "kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/ingress.yaml"
                sh "kubectl apply -f https://raw.githubusercontent.com/appscode/k8s-addons/master/api/extensions/certificate.yaml"
                sh "echo $deployment_yaml | kubectl create -f -"
            }
            stage("e2e test" {
                sh "./hack/make.py test e2e -cloud-provider=$cloud_provier -cluster-name=$cluster_name"
            })
        }
        currentBuild.result = 'SUCCESS'
    } catch (Exception err) {
        currentBuild.result = 'FAILURE'
    } finally {
        deleteDir()
        sh "docker rmi -f $IMAGE:$INTERNAL_TAG"
    }
}
