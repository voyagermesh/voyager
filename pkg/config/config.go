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

package config

import (
	"time"

	"kmodules.xyz/client-go/meta"
)

var (
	BuiltinTemplates = "/srv/voyager/templates/*.cfg"
)

func init() {
	if meta.PossiblyInCluster() {
		BuiltinTemplates = "/srv/voyager/templates/*.cfg"
	} else {
		BuiltinTemplates = "/src/hack/docker/voyager/templates/*.cfg"
		// BuiltinTemplates = "/home/tamal/go/src/voyagermesh.dev/voyager/hack/docker/voyager/templates/*.cfg"
	}
}

type Config struct {
	Burst                       int
	CloudConfigFile             string
	CloudProvider               string
	HAProxyImage                string
	ExporterImage               string
	IngressClass                string
	MaxNumRequeues              int
	NumThreads                  int
	OperatorNamespace           string
	OperatorService             string
	QPS                         float32
	RestrictToOperatorNamespace bool
	ResyncPeriod                time.Duration
	WatchNamespace              string
	ValidateHAProxyConfig       bool
	EnableValidatingWebhook     bool
}
