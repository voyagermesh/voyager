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

package framework

import (
	"net"

	"gomodules.xyz/cert"
	"gomodules.xyz/x/crypto/rand"
)

func (ri *rootInvocation) UniqueName() string {
	return rand.WithUniqSuffix("e2e-test")
}

func (f *Framework) ServerSANs() cert.AltNames {
	return cert.AltNames{
		IPs:      []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.99.100")},
		DNSNames: []string{"server", TestDomain},
	}
}

var (
	ClientOrgs = []string{"AppsCode", "Eng"}
)
