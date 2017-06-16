package testserverclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/appscode/log"
	"github.com/stretchr/testify/assert"
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
	log.Infoln("Handling HTTP Request")
	json.NewEncoder(w).Encode(resp)
}
