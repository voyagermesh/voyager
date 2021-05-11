/*
Copyright AppsCode Inc. and Contributors

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

package client

import (
	"fmt"
	"net"
	"testing"

	proxyproto "github.com/pires/go-proxyproto"
)

const (
	NO_PROTOCOL = "There is no spoon"
	IP4_ADDR    = "127.0.0.1"
	IP6_ADDR    = "::1"
	PORT        = 65533
)

var (
	v4addr = net.ParseIP(IP4_ADDR).To4()
	v6addr = net.ParseIP(IP6_ADDR).To16()
)

func TestTCPServer(t *testing.T) {
	resp, err := NewTestTCPClient("127.0.0.1:6767").DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}

func TestProxyV1Server(t *testing.T) {
	resp, err := NewTestTCPClient("127.0.0.1:6767").
		WithProxyHeader(&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   v4addr,
				Port: PORT,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   v4addr,
				Port: PORT,
			},
		}).DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}

func TestProxyV2Server(t *testing.T) {
	resp, err := NewTestTCPClient("127.0.0.1:6767").
		WithProxyHeader(&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   v4addr,
				Port: PORT,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   v4addr,
				Port: PORT,
			},
		}).DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}
