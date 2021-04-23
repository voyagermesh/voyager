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
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetTimeOuts(t *testing.T) {
	ing := &Ingress{}
	timeouts := ing.Timeouts()

	assert.Equal(t, timeouts, timeoutDefaults)
}

func TestGetOptions(t *testing.T) {
	ing := &Ingress{
		ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
			"ingress.appscode.com/default-option": `{"positive-options": "true", "negative-options": "false", "to-be-ignored": ""}`,
		}},
	}
	opt := ing.HAProxyOptions()
	responseMap := map[string]bool{
		"positive-options": true,
		"negative-options": false,
	}
	assert.Equal(t, responseMap, opt)
}
