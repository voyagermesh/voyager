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

package prometheus

import (
	kutil "kmodules.xyz/client-go"
	api "kmodules.xyz/monitoring-agent-api/api/v1"
)

// Prometheus agent does nothing as user is expected to setup Prometheus manually for scraping.
// xref: https://github.com/kmodules/monitoring-agent-api/issues/25#issuecomment-567635963
type Prometheus struct {
}

func New() api.Agent {
	return &Prometheus{}
}

func (agent *Prometheus) GetType() api.AgentType {
	return api.AgentPrometheus
}

func (agent *Prometheus) CreateOrUpdate(sp api.StatsAccessor, new *api.AgentSpec) (kutil.VerbType, error) {
	return kutil.VerbUnchanged, nil
}

func (agent *Prometheus) Delete(sp api.StatsAccessor) (kutil.VerbType, error) {
	return kutil.VerbUnchanged, nil
}
