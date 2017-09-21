// Copyright 2013 Xing Xing <mikespook@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a commercial
// license that can be found in the LICENSE file.

package idgen

import (
	"github.com/mikespook/golib/autoinc"
	"gopkg.in/mgo.v2/bson"
)

type IdGen interface {
	Id() interface{}
}

// ObjectId
type objectId struct{}

func (id *objectId) Id() interface{} {
	return bson.NewObjectId().Hex()
}

func NewObjectId() *objectId {
	return &objectId{}
}

// AutoIncId
type autoincId struct {
	*autoinc.AutoInc
}

func (id *autoincId) Id() interface{} {
	return id.AutoInc.Id()
}

func NewAutoIncId() *autoincId {
	return &autoincId{autoinc.New(1, 1)}
}
