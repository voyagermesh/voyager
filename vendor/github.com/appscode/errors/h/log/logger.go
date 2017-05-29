package log

import (
	"github.com/appscode/log"
)

type LogHandler struct{}

func (LogHandler) Handle(e error) {
	if e != nil {
		log.Error("Error:", e.Error())
	}
}
