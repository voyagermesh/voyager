package log

import (
	"github.com/appscode/errors"
	"github.com/appscode/log"
)

type LogHandler struct{}

func (LogHandler) Handle(e error) {
	if e != nil {
		if err := errors.FromErr(e); err != nil {
			if err.Err() != nil {
				log.Error("Error:", err.Err().Error())
			}
			if len(err.Message()) > 0 {
				log.Error(err.Message())
			}
			if err.Context() != nil {
				log.Error(err.Context().String())
			}
			log.Error("Trace\n", err.TraceString(), "\n")
		} else {
			log.Error("Error:", e.Error())
		}
	}
}
