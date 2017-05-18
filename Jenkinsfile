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
        try {
            sh "go version"
        } catch (e) {
            sh "sudo apt update && sudo apt install -y curl &&\
          curl -OL https://storage.googleapis.com/golang/go${go_version}.linux-amd64.tar.gz &&\
          tar -xzf go${go_version}.linux-amd64.tar.gz &&\
          rm -rf go${go_version}.linux-amd64.tar.gz"
        }
    }
    //TODO: @ashiq
    //content builddeps should place to ./hack/builddeps.sh
    stage('builddeps') {
        sh 'sudo apt update &&\
        sudo apt install -y software-properties-common python-software-properties python-dev libyaml-dev python-pip build-essential &&\
        pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage &&\
        go get -u golang.org/x/tools/cmd/goimports &&\
        go get -u github.com/sgotti/glide-vc &&\
        curl https://glide.sh/get | sh'
    }
    dir("${project_dir}") {
        stage("checkout") {
            checkout scm
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
    }
}
