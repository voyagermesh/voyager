/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"sync/atomic"

	"k8s.io/klog/v2"
)

var (
	reloadPerformed uint64
	configChanged   uint64
	certChanged     uint64
)

func incConfigChangedCounter() {
	atomic.AddUint64(&configChanged, 1)
	klog.Infoln("config changed:", atomic.LoadUint64(&configChanged))
}

func incCertChangedCounter() {
	atomic.AddUint64(&certChanged, 1)
	klog.Infoln("cert changed:", atomic.LoadUint64(&certChanged))
}

func incReloadCounter() {
	atomic.AddUint64(&reloadPerformed, 1)
	klog.Infoln("HAProxy reloaded:", atomic.LoadUint64(&reloadPerformed))
}
