/*
Copyright The Kmodules Authors.

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
	jsoniter "github.com/json-iterator/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var json = jsoniter.ConfigFastest

func IsOwnedByDeployment(refs []metav1.OwnerReference) bool {
	for _, ref := range refs {
		if ref.Kind == "Deployment" && ref.Name != "" {
			return true
		}
	}
	return false
}
