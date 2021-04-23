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
	"encoding/json"

	core "k8s.io/api/core/v1"
)

func NewEngressFromIngress(ing interface{}) (*Ingress, error) {
	data, err := json.Marshal(ing)
	if err != nil {
		return nil, err
	}
	r := &Ingress{}
	err = json.Unmarshal(data, r)
	if err != nil {
		return nil, err
	}
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[APISchema] = APISchemaIngress

	if v, _ := get[NodeSelector](r.Annotations); v != nil {
		r.Spec.NodeSelector = v.(map[string]string)
	}
	delete(r.Annotations, NodeSelector)

	if v, _ := get[Tolerations](r.Annotations); v != nil {
		r.Spec.Tolerations = v.([]core.Toleration)
	}
	delete(r.Annotations, Tolerations)

	r.Migrate()
	return r, nil
}
