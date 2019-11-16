/*
Copyright The Voyager Authors.

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

package providers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNotFound(t *testing.T) {
	ts := httptest.NewServer(DefaultHTTPProvider().NewServeMux())
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("invalid test server url %s", err)
	}

	u.Path = "/.well-known/acme-challenge/token"
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal("expected Nil, found", err)
	}

	if resp.StatusCode != 200 {
		t.Fatal("expected 200, found", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("expected Nil, found", err)
	}

	if string(data) != "TEST" {
		t.Fatal("response do not matched")
	}
}

func TestFound(t *testing.T) {
	ts := httptest.NewServer(DefaultHTTPProvider().NewServeMux())
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("invalid test server url %s", err)
	}

	defaultHTTPProvider.Present(u.Host, "token", "key")

	u.Path = "/.well-known/acme-challenge/token"
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal("expected Nil, found", err)
	}

	if resp.StatusCode != 200 {
		t.Fatal("expected 200, found", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("expected Nil, found", err)
	}

	if string(data) != "key" {
		t.Fatal("response do not matched, got", string(data))
	}
}
