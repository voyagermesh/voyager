package log

import (
	"github.com/appscode/errors"
)

type LogHandler struct{}

func (LogHandler) Handle(e error) {
	errors.Log(e)
}
