package client

import (
	"fmt"
	"net"
	"testing"

	"github.com/pires/go-proxyproto"
)

const (
	NO_PROTOCOL = "There is no spoon"
	IP4_ADDR    = "127.1.0.1"
	IP6_ADDR    = "::1"
	PORT        = 65533
)

var (
	v4addr = net.ParseIP(IP4_ADDR).To4()
	v6addr = net.ParseIP(IP6_ADDR).To16()
)

func TestTCPServer(t *testing.T) {
	resp, err := NewTestTCPClient("127.1.0.1:6767").DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}

func TestProxyV1Server(t *testing.T) {
	resp, err := NewTestTCPClient("127.1.0.1:6767").
		WithProxyHeader(&proxyproto.Header{
			Version:            1,
			Command:            proxyproto.PROXY,
			TransportProtocol:  proxyproto.TCPv4,
			SourceAddress:      v4addr,
			DestinationAddress: v4addr,
			SourcePort:         PORT,
			DestinationPort:    PORT,
		}).DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}

func TestProxyV2Server(t *testing.T) {
	resp, err := NewTestTCPClient("127.1.0.1:6767").
		WithProxyHeader(&proxyproto.Header{
			Version:            2,
			Command:            proxyproto.PROXY,
			TransportProtocol:  proxyproto.TCPv4,
			SourceAddress:      v4addr,
			DestinationAddress: v4addr,
			SourcePort:         PORT,
			DestinationPort:    PORT,
		}).DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}
