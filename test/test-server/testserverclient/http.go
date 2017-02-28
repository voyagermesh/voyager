package testserverclient

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type httpClient struct {
	client  *http.Client
	baseURL string
	method  string
	path    string
	header  map[string]string
}

type Response struct {
	Status     int         `json:"-"`
	Type       string      `json:"type,omitempty"`
	Host       string      `json:"host,omitempty"`
	ServerPort string      `json:"serverPort,omitempty"`
	Path       string      `json:"path,omitempty"`
	Method     string      `json:"method,omitempty"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       string      `json:"body,omitempty"`
}

func NewTestHTTPClient(url string) *httpClient {
	url = strings.TrimSuffix(url, "/")
	return &httpClient{
		client:  &http.Client{Timeout: time.Second * 5},
		baseURL: url,
	}
}

func (t *httpClient) Method(method string) *httpClient {
	t.method = method
	return t
}

func (t *httpClient) Path(path string) *httpClient {
	path = strings.TrimPrefix(path, "/")
	t.path = path
	return t
}

func (t *httpClient) Header(h map[string]string) *httpClient {
	t.header = h
	return t
}

func (t *httpClient) Do() (*Response, error) {
	req, err := http.NewRequest(t.method, t.baseURL+"/"+t.path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range t.header {
		req.Header.Add(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	responseStruct := &Response{
		Status: resp.StatusCode,
	}
	err = json.NewDecoder(resp.Body).Decode(responseStruct)
	if err != nil {
		return nil, err
	}

	return responseStruct, nil
}
