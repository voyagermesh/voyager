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
    stage('builddeps') {
        sh 'sudo apt update &&\
        sudo apt install -y software-properties-common python-software-properties python-dev libyaml-dev python-pip build-essential libsqlite3-dev &&\
        sudo pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage &&\
        sudo apt install curl'
    }
    stage("go setup") {
        try {
            sh "go version"
        } catch (e) {
            sh "curl -OL https://storage.googleapis.com/golang/go${go_version}.linux-amd64.tar.gz &&\
          tar -xzf go${go_version}.linux-amd64.tar.gz &&\
          rm -rf go${go_version}.linux-amd64.tar.gz"
        }
    }
    stage("dependency") {
        sh "go get -u github.com/jteeuwen/go-bindata &&\
          go install github.com/jteeuwen/go-bindata/... &&\
          go get -u github.com/progrium/go-extpoints &&\
          go install github.com/progrium/go-extpoints &&\
          go get -u golang.org/x/tools/cmd/goimports &&\
          go install golang.org/x/tools/cmd/goimports"
    }
    stage("install glide") {
        sh "curl https://glide.sh/get | sh"
    }
    stage("checkout") {
        dir("${project_dir}") {
            checkout scm
        }
    }
    stage("build binary") {
        dir("${project_dir}") {
            sh "glide slow"
            sh "./hack/make.py"
        }
    }
    stage("build docker") {
        dir("${project_dir}") {
            sh "./hack/docker/voyager/setup.sh"
        }
    }
    stage("docker push") {

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
