package testserverclient

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/log"
	"github.com/moul/http2curl"
)

type httpClient struct {
	client  *http.Client
	baseURL string
	method  string
	path    string
	host    string
	header  map[string]string
}

type Response struct {
	Status          int         `json:"-"`
	ResponseHeader  http.Header `json:"-"`
	Type            string      `json:"type,omitempty"`
	PodName         string      `json:"podName,omitempty"`
	Host            string      `json:"host,omitempty"`
	ServerPort      string      `json:"serverPort,omitempty"`
	Path            string      `json:"path,omitempty"`
	Method          string      `json:"method,omitempty"`
	RequestHeaders  http.Header `json:"headers,omitempty"`
	Body            string      `json:"body,omitempty"`
	HTTPSServerName string      `json:"-"`
}

func NewTestHTTPClient(url string) *httpClient {
	url = strings.TrimSuffix(url, "/")
	return &httpClient{
		client:  &http.Client{Timeout: time.Second * 5},
		baseURL: url,
	}
}

func (t *httpClient) WithCert(cert string) *httpClient {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(cert))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	t.client.Transport = tr
	return t
}

func (t *httpClient) WithInsecure() *httpClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	t.client.Transport = tr
	return t
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

func (t *httpClient) WithHost(host string) *httpClient {
	t.host = host
	return t
}

func (t *httpClient) Header(h map[string]string) *httpClient {
	t.header = h
	return t
}

func (t *httpClient) DoWithRetry(limit int) (*Response, error) {
	var resp *Response
	var err error
	for i := 1; i <= limit; i++ {
		resp, err = t.do(true)
		if err == nil {
			return resp, err
		}
		time.Sleep(time.Second * 5)
	}
	return resp, err
}

func (t *httpClient) DoTestRedirectWithRetry(limit int) (*Response, error) {
	var resp *Response
	var err error

	// Do Not redirect
	t.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for i := 1; i <= limit; i++ {
		resp, err = t.do(false)
		if err == nil {
			return resp, err
		}
		time.Sleep(time.Second * 5)
	}
	return resp, err
}

func (t *httpClient) DoStatusWithRetry(limit int) (*Response, error) {
	var resp *Response
	var err error
	for i := 1; i <= limit; i++ {
		resp, err = t.do(false)
		if err == nil {
			return resp, err
		}
		time.Sleep(time.Second * 5)
	}
	return resp, err
}

func (t *httpClient) do(parse bool) (*Response, error) {
	req, err := http.NewRequest(t.method, t.baseURL+"/"+t.path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range t.header {
		req.Header.Add(k, v)
	}

	if len(t.host) > 0 {
		req.Header.Add("Host", t.host)
		req.Host = t.host
	}

	if val := req.Header.Get("Content-Length"); len(val) > 0 {
		cl, _ := strconv.Atoi(val)
		req.ContentLength = int64(cl)
		req.Body = newBody(cl)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	req.Body = nil
	command, _ := http2curl.GetCurlCommand(req)
	log.Warningln("Request:", command)

	responseStruct := &Response{
		Status:         resp.StatusCode,
		ResponseHeader: resp.Header,
	}

	if t.client.Transport != nil {
		if resp.TLS != nil {
			responseStruct.HTTPSServerName = resp.TLS.ServerName
		}
	}

	if parse {
		err = json.NewDecoder(resp.Body).Decode(responseStruct)
		if err != nil {
			return nil, err
		}
	}
	return responseStruct, nil
}

func newBody(size int) io.ReadCloser {
	r := make([]byte, size)
	rand.Read(r)
	return &nopCloser{
		Reader: bytes.NewReader(r),
	}
}

type nopCloser struct {
	io.Reader
}

func (*nopCloser) Close() error { return nil }
