package controller

import (
	"sync/atomic"

	"github.com/appscode/go/log"
)

var (
	reloadPerformed uint64
	configChanged   uint64
	certChanged     uint64
)

func incConfigChangedCounter() {
	atomic.AddUint64(&configChanged, 1)
	log.Infoln("config changed:", atomic.LoadUint64(&configChanged))
}

func incCertChangedCounter() {
	atomic.AddUint64(&certChanged, 1)
	log.Infoln("cert changed:", atomic.LoadUint64(&certChanged))
}

func incReloadCounter() {
	atomic.AddUint64(&reloadPerformed, 1)
	log.Infoln("HAProxy reloaded:", atomic.LoadUint64(&reloadPerformed))
}
