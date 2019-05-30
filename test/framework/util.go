package framework

import (
	"net"

	"github.com/appscode/go/crypto/rand"
	"gomodules.xyz/cert"
)

func (r *rootInvocation) UniqueName() string {
	return rand.WithUniqSuffix("e2e-test")
}

func (r *Framework) ServerSANs() cert.AltNames {
	return cert.AltNames{
		IPs:      []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.99.100")},
		DNSNames: []string{"server", TestDomain},
	}
}

var (
	ClientOrgs = []string{"AppsCode", "Eng"}
)
