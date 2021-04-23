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

package v1beta1

import (
	"fmt"

	mona "kmodules.xyz/monitoring-agent-api/api/v1"
)

const (
	StatsPortName             = "stats"
	ExporterPortName          = "http"
	DefaultExporterPortNumber = 56790
)

func (r Ingress) StatsAccessor() mona.StatsAccessor {
	return &statsService{ing: r}
}

type statsService struct {
	ing Ingress
}

func (s statsService) ServiceMonitorAdditionalLabels() map[string]string {
	return s.ing.OffshootLabels()
}

func (s statsService) GetNamespace() string {
	return s.ing.Namespace
}

func (s statsService) ServiceName() string {
	return s.ing.StatsServiceName()
}

func (s statsService) ServiceMonitorName() string {
	return s.ServiceName()
}

func (s statsService) Path() string {
	return fmt.Sprintf("/%s/namespaces/%s/ingresses/%s/metrics", s.ing.APISchema(), s.ing.Namespace, s.ing.Name)
}

func (s statsService) Scheme() string {
	return ""
}
