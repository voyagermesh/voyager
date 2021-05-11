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
	core "k8s.io/api/core/v1"
)

func (r Ingress) ObjectReference() *core.ObjectReference {
	if r.APISchema() == APISchemaIngress {
		return &core.ObjectReference{
			APIVersion:      APISchemaIngress,
			Kind:            ResourceKindIngress,
			Namespace:       r.Namespace,
			Name:            r.Name,
			UID:             r.UID,
			ResourceVersion: r.ResourceVersion,
		}
	}
	return &core.ObjectReference{
		APIVersion:      SchemeGroupVersion.String(),
		Kind:            ResourceKindIngress,
		Namespace:       r.Namespace,
		Name:            r.Name,
		UID:             r.UID,
		ResourceVersion: r.ResourceVersion,
	}
}
