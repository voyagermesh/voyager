node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    def go_version = "1.8.1"
    stage("set env") {
        env.GOROOT = "${PWD}/go"
        env.GOPATH = "${PWD}"
        env.GOBIN = "${GOPATH}/bin"
        env.PATH = "${env.GOROOT}/bin:${env.PATH}:$GOPATH:$GOBIN"
        sh "mkdir -p ${env.GOBIN}"
    }
    stage("go setup") {
        sh "sudo apt update && sudo apt-get -y install curl"
        try {
            sh "go version"
        } catch (e) {
            sh "curl -OL https://storage.googleapis.com/golang/go${go_version}.linux-amd64.tar.gz &&\
          tar -xzf go${go_version}.linux-amd64.tar.gz &&\
          rm -rf go${go_version}.linux-amd64.tar.gz"
        }
    }
    dir("${project_dir}") {
        stage("checkout") {
            checkout scm
        }
        stage('builddeps') {
            sh "go version"
            sh "sudo ./hack/builddeps.sh"
        }
        stage("build binary") {
            sh "glide slow"
            sh "./hack/make.py"
        }
        stage("build docker") {
            sh "./hack/docker/voyager/setup.sh"
        }
    }
/*    stage("test") {
        dir("${project_dir}") {
            sh "mkdir ~/.kube && cp /srv/appscode/comissionar/config ~/.kube/config"
            sh "./hack/make.py test integration --cloud-provider=gce --cluster-name=apscd"
        }
    }*/
/*    post {
        always {
            dir("${project_dir}") {
                deleteDir()
            }
        }
    }*/
}
