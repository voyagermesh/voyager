#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscode/voyager/hack/gendocs
go run main.go

cd $GOPATH/src/github.com/appscode/voyager/docs/reference
sed -i 's/######\ Auto\ generated\ by.*//g' *
popd
