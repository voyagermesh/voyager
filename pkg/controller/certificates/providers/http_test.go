package providers

import (
	"io/ioutil"
	"net/http"
	"testing"

	flags "github.com/appscode/go-flags"
)

func init() {
	flags.SetLogLevel(10)
}

func TestNotFound(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:8080" + URLPrefix + "token")
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
	defaultHTTPProvider.Present("127.0.0.1:8080", "token", "key")
	resp, err := http.Get("http://127.0.0.1:8080" + URLPrefix + "token")
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
