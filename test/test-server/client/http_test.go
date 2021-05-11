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

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func TestNewTestHTTPClient(t *testing.T) {
	sv := httptest.NewServer(testHTTPHandler{})
	resp, err := NewTestHTTPClient(sv.URL).Method("GET").Path("/hello/world").DoWithRetry(1)
	if assert.Nil(t, err) {
		assert.Equal(t, resp.Method, "GET")
		assert.Equal(t, resp.Path, "/hello/world")
	}
}

type testHTTPHandler struct {
	port string
}

func (h testHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := &Response{
		Type:           "http",
		Host:           r.Host,
		ServerPort:     h.port,
		Path:           r.URL.Path,
		Method:         r.Method,
		RequestHeaders: r.Header,
	}
	klog.Infoln("Handling HTTP Request")
	json.NewEncoder(w).Encode(resp)
}
