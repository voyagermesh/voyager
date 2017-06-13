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
            'linux': ['amd64']
        }
    }
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
    libbuild.ungroup_go_imports('*.go', 'api', 'client', 'pkg', 'test')
    die(call('goimports -w *.go api client pkg test'))
    call('gofmt -s -w *.go api client pkg test')


def vet():
    call('go vet *.go ./api/... ./client/... ./pkg/... ./test/...')


def lint():
    call('golint *.go ./api/... ./client/... ./pkg/... ./test/...')


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


def build_cmds():
    gen()
    for name in libbuild.BIN_MATRIX:
        build_cmd(name)


def build(name=None):
    if name:
        cfg = libbuild.BIN_MATRIX[name]
        if cfg['type'] == 'go':
            gen()
            build_cmd(name)
    else:
        build_cmds()


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
    die(call('GO15VENDOREXPERIMENT=1 ' + libbuild.GOC + ' install ./cmd/...'))


def test(type, *args):
    die(call(libbuild.GOC + ' install ./cmd/...'))
    pydotenv.load_dotenv(join(dirname(__file__), 'configs/.env'))
    if type == 'unit':
        unit_test()
    elif type == 'e2e':
        e2e_test(args)
    elif type == 'minikube':
        e2e_test_minikube(args)
    elif type == 'integration' or type == 'intg':
        integration(args)
    elif type == 'clean':
        e2e_test_clean()
    else:
        print '{test unit|minikube|e2e}'

def unit_test():
    die(call(libbuild.GOC + ' test -v ./cmd/... ./pkg/... -args -v=3 -verbose=true -mode=unit'))

def e2e_test(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/e2e/... -timeout 10h -args -v=3 -verbose=true -mode=e2e ' + st))

def e2e_test_minikube(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/e2e/... -timeout 10h -args -v=3 -verbose=true -mode=e2e -cloud-provider=minikube ' + st))

def integration(args):
    st = ' '.join(args)
    die(call(libbuild.GOC + ' test -v ./test/integration/... -timeout 10h -args -v=3 -verbose=true -mode=e2e -in-cluster=true ' + st))

def testd():
    print yaml.dump({'gold': 10, 'sdfsfds': 34}, default_flow_style=True)
    with open('/home/tamal/go/src/github.com/appscode/voyager/hack/deploy/deployments.yaml', 'r') as f:
        docs = yaml.load_all(f)
        result = []
        for doc in docs:
            if doc['kind'] == 'Deployment':
                c = doc['spec']['template']['spec']['containers'][0]
                c['image'] = 'appscode/voyager:xyzxyz'
                c['args'] = [
                    'run',
                    '--cloud-provider=gce',
                    '--v=3',
                    '--analytics=false'
                ]
            result.append(doc)
        print yaml.dump_all(result, default_flow_style=False)

def default():
    gen()
    fmt()
    die(call('GO15VENDOREXPERIMENT=1 ' + libbuild.GOC + ' install .'))


if __name__ == "__main__":
    if len(sys.argv) > 1:
        # http://stackoverflow.com/a/834451
        # http://stackoverflow.com/a/817296
        globals()[sys.argv[1]](*sys.argv[2:])
    else:
        default()
