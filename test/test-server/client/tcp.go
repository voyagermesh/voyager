package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"github.com/pires/go-proxyproto"
)

type tcpClient struct {
	url    string
	ssl    bool
	cert   string
	header *proxyproto.Header
}

func NewTestTCPClient(url string) *tcpClient {
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}
	return &tcpClient{
		url: url,
	}
}

func (t *tcpClient) WithProxyHeader(header *proxyproto.Header) *tcpClient {
	t.header = header
	return t
}

func (t *tcpClient) WithSSL(cert string) *tcpClient {
	t.ssl = true
	t.cert = cert
	return t
}

func (t *tcpClient) DoWithRetry(limit int) (*Response, error) {
	var resp *Response
	var err error
	for i := 1; i <= limit; i++ {
		resp, err = t.do()
		if err == nil {
			return resp, err
		}
		time.Sleep(time.Second * 5)
	}
	return resp, err
}

func (t *tcpClient) do() (*Response, error) {
	conn, err := net.Dial("tcp", t.url)
	if err != nil {
		return nil, err
	}

	if t.ssl {
		config := &tls.Config{}
		if len(t.cert) > 0 {
			roots := x509.NewCertPool()
			roots.AppendCertsFromPEM([]byte(t.cert))
			config = &tls.Config{RootCAs: roots}
		} else {
			config = &tls.Config{InsecureSkipVerify: true}
		}
		conn = tls.Client(conn, config)
	}

	if t.header != nil {
		t.header.WriteTo(conn)
	}

	var buf bytes.Buffer
	io.Copy(&buf, conn)
	data := buf.String()

	req := &Response{}

	if len(data) <= 2 || (data[0] != '{') {
		req.Body = data
		return req, nil
	}

	err = json.NewDecoder(strings.NewReader(string(data))).Decode(req)
	if err != nil {
		return nil, err
	}
	return req, nil
}
