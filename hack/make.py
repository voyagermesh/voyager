#!/usr/bin/env python


# http://stackoverflow.com/a/14050282
def check_antipackage():
    from sys import version_info
    sys_version = version_info[:2]
    found = True
    if sys_version < (3, 0):
        # 'python 2'
        from pkgutil import find_loader
        found = find_loader('antipackage') is not None
    elif sys_version <= (3, 3):
        # 'python <= 3.3'
        from importlib import find_loader
        found = find_loader('antipackage') is not None
    else:
        # 'python >= 3.4'
        from importlib import util
        found = util.find_spec('antipackage') is not None
    if not found:
        print('Install missing package "antipackage"')
        print('Example: pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage')
        from sys import exit
        exit(1)
check_antipackage()

# ref: https://github.com/ellisonbg/antipackage
import antipackage
from github.appscode.libbuild import libbuild, pydotenv

import os
import os.path
import subprocess
import sys
import yaml
from os.path import expandvars, join, dirname

libbuild.REPO_ROOT = expandvars('$GOPATH') + '/src/github.com/appscode/voyager'
BUILD_METADATA = libbuild.metadata(libbuild.REPO_ROOT)
libbuild.BIN_MATRIX = {
    'voyager': {
        'type': 'go',
        'go_version': True,
        'use_cgo': False,
        'distro': {
            'alpine': ['amd64'],
            'darwin': ['amd64'],
            'linux': ['amd64']
        }
    }
}
if libbuild.ENV not in ['prod']:
    libbuild.BIN_MATRIX['voyager']['distro'] = {
        'alpine': ['amd64']
    }
libbuild.BUCKET_MATRIX = {
    'prod': 'gs://appscode-cdn',
    'dev': 'gs://appscode-dev'
}


def call(cmd, stdin=None, cwd=libbuild.REPO_ROOT):
    print(cmd)
    return subprocess.call([expandvars(cmd)], shell=True, stdin=stdin, cwd=cwd)


def die(status):
    if status:
        sys.exit(status)


def check_output(cmd, stdin=None, cwd=libbuild.REPO_ROOT):
    print(cmd)
    return subprocess.check_output([expandvars(cmd)], shell=True, stdin=stdin, cwd=cwd)


def version():
    # json.dump(BUILD_METADATA, sys.stdout, sort_keys=True, indent=2)
    for k in sorted(BUILD_METADATA):
        print(k + '=' + BUILD_METADATA[k])


def fmt():
    libbuild.ungroup_go_imports('*.go', 'apis', 'client', 'pkg', 'test', 'third_party')
    die(call('goimports -w *.go apis client pkg test third_party'))
    call('gofmt -s -w *.go apis client pkg test third_party')


def vet():
    call('go vet *.go ./apis/... ./client/... ./pkg/... ./test/...')


def lint():
    call('golint *.go ./apis/... ./client/... ./pkg/... ./test/...')


def gen():
    return


def build_cmd(name):
    cfg = libbuild.BIN_MATRIX[name]
    if cfg['type'] == 'go':
        if 'distro' in cfg:
            for goos, archs in cfg['distro'].items():
                for goarch in archs:
                    libbuild.go_build(name, goos, goarch, main='*.go')
        else:
            libbuild.go_build(name, libbuild.GOHOSTOS, libbuild.GOHOSTARCH, main='*.go')


def build(name=None):
    gen()
    fmt()
    if name:
        cfg = libbuild.BIN_MATRIX[name]
        if cfg['type'] == 'go':
            build_cmd(name)
    else:
        for name in libbuild.BIN_MATRIX:
            build_cmd(name)


def push(name=None):
    if name:
        bindir = libbuild.REPO_ROOT + '/dist/' + name
        push_bin(bindir)
    else:
        dist = libbuild.REPO_ROOT + '/dist'
        for name in os.listdir(dist):
            d = dist + '/' + name
            if os.path.isdir(d):
                push_bin(d)


def push_bin(bindir):
    call('rm -f *.md5', cwd=bindir)
    call('rm -f *.sha1', cwd=bindir)
    for f in os.listdir(bindir):
        if os.path.isfile(bindir + '/' + f):
            libbuild.upload_to_cloud(bindir, f, BUILD_METADATA['version'])


def update_registry():
    libbuild.update_registry(BUILD_METADATA['version'])


def install():
    die(call(libbuild.GOC + ' install ./...'))


def test(type, *args):
    die(call(libbuild.GOC + ' install ./...'))

    if os.path.exists(libbuild.REPO_ROOT + "/hack/configs/.env"):
        print 'Loading env file'
        pydotenv.load_dotenv(libbuild.REPO_ROOT + "/hack/configs/.env")

    if type == 'unit':
        unit_test(args)
    elif type == 'e2e':
        e2e_test(args)
    elif type == 'minikube':
        e2e_test_minikube(args)
    elif type == 'integration' or type == 'int':
        integration_test(args)
    else:
        print '{test unit|minikube|e2e}'

def unit_test(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v . ./api/... ./client/... ./pkg/... ' + st))

def e2e_test(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/e2e/... -timeout 10h -args -ginkgo.v -ginkgo.progress -ginkgo.trace -v=2 ' + st))

def e2e_test_minikube(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/e2e/... -timeout 10h -args -ginkgo.v -ginkgo.progress -ginkgo.trace -v=2 -cloud-provider=minikube ' + st))

def integration_test(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/e2e/... -timeout 10h -args -ginkgo.v -ginkgo.progress -ginkgo.trace -v=2 -in-cluster=true ' + st))

def test_deploy(provider):
    with open(libbuild.REPO_ROOT + '/hack/deploy/deployments.yaml', 'r') as f:
        docs = yaml.load_all(f)
        result = []
        for doc in docs:
            if doc['kind'] == 'Deployment':
                c = doc['spec']['template']['spec']['containers'][0]
                c['image'] = 'appscode/voyager:' + BUILD_METADATA['version']
                c['args'] = [
                    'run',
                    '--cloud-provider=' + provider,
                    '--v=5',
                    '--analytics=false'
                ]
            result.append(doc)
        dist = libbuild.REPO_ROOT + '/dist'
        if not os.path.exists(dist):
            os.makedirs(dist)
        with file(dist + '/kube.yaml', 'w') as out:
            yaml.dump_all(result, out, default_flow_style=False)


def default():
    gen()
    fmt()
    die(call('GOBIN={} {} install . ./test/...'.format(libbuild.GOBIN, libbuild.GOC)))


if __name__ == "__main__":
    if len(sys.argv) > 1:
        # http://stackoverflow.com/a/834451
        # http://stackoverflow.com/a/817296
        globals()[sys.argv[1]](*sys.argv[2:])
    else:
        default()
