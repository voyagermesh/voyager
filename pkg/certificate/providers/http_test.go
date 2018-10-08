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
