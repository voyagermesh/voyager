package providers

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
	"github.com/appscode/voyager/test/testframework"
)

func init() {
	testframework.Initialize()
}

func TestNotFound(t *testing.T) {
	defaultHTTPProvider.serve()
	time.Sleep(time.Second * 5)
	resp, err := http.Get("http://127.0.0.1:56789" + URLPrefix + "token")
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
	defaultHTTPProvider.serve()
	time.Sleep(time.Second * 5)
	defaultHTTPProvider.Present("127.0.0.1:56789", "token", "key")
	resp, err := http.Get("http://127.0.0.1:56789" + URLPrefix + "token")
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
