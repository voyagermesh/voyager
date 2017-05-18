node("master") {
    def PWD = pwd()
    def project_dir = "${PWD}/src/github.com/appscode/voyager"
    stage("set env") {
        env.GOPATH = "${PWD}"
        env.GOBIN = "${GOPATH}/bin"
        env.PATH = "$env.PATH:${env.GOBIN}"
        sh "mkdir -p ${env.GOBIN}"
    }
    dir("${project_dir}") {
        stage("checkout") {
            checkout scm
        }
        stage("builddeps") {
            sh "./hack/builddeps.sh"
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
