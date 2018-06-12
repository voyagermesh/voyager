package framework

import (
	"net"

	"github.com/appscode/go/crypto/rand"
	"k8s.io/client-go/util/cert"
)

func (r *rootInvocation) UniqueName() string {
	return rand.WithUniqSuffix("e2e-test")
}

func (r *Framework) ServerSANs() cert.AltNames {
	return cert.AltNames{
		IPs:      []net.IP{net.ParseIP("127.1.0.1"), net.ParseIP("192.168.99.100")},
		DNSNames: []string{TestDomain},
	}
}

var (
	ClientOrgs = []string{"AppsCode", "Eng"}
)
