package providers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestNotFound(t *testing.T) {
	defaultHTTPProvider.serve()
	time.Sleep(time.Second * 5)
	resp, err := http.Get(fmt.Sprintf("http://127.1.0.1:%d/.well-known/acme-challenge/token", ACMEResponderPort))
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
	defaultHTTPProvider.Present(fmt.Sprintf("127.1.0.1:%d", ACMEResponderPort), "token", "key")
	resp, err := http.Get(fmt.Sprintf("http://127.1.0.1:%d/.well-known/acme-challenge/token", ACMEResponderPort))
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
