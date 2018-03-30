#!/bin/sh

set -x -e

mkdir -p $GOPATH/src/github.com/appscode
cp -r voyager $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/voyager/hack
./make.py test unit
