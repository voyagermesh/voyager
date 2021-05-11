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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func TestValidateConfig(t *testing.T) {
	data := map[string]bool{
		`
# global
# 	lua-load /etc/auth-request.lua
listen http-in
	bind *:80
	server http-server 127.0.0.1:8080`: true, // valid config - required /etc/auth-request.lua

		`listen http-in`: false, // invalid config - no bind

		``: false, // invalid config - no listener
	}

	for cfg, result := range data {
		err := ValidateConfig(cfg)
		if err != nil {
			klog.Errorln(err)
		}
		assert.Equal(t, result, err == nil)
	}
}
