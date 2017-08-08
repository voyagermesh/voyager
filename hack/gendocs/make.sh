#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscode/voyager/hack/gendocs
go run main.go
popd
