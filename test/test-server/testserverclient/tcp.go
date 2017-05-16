package testserverclient

import (
	"encoding/json"
	"net"
	"time"
	"strings"
)

type tcpClient struct {
	url string
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

	resp := &Response{}
	err = json.NewDecoder(conn).Decode(resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
