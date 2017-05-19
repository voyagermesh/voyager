node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    stage("set env") {
        env.GOPATH = "${PWD}"
        env.GOBIN = "${GOPATH}/bin"
        env.PATH = "$env.PATH:${env.GOBIN}:/usr/local/go/bin"
        sh "mkdir -p ${env.GOBIN}"
    }
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
    }
}
