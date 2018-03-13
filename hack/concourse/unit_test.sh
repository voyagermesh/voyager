#!/bin/sh

set -x -e

ROOT_DIR=$(pwd)
mkdir -p $GOPATH/src/github.com/appscode
cp -r voyager $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/voyager/hack
./make.py test unit
