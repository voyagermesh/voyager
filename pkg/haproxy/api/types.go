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

package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
)

type TemplateData struct {
	*SharedInfo
	TimeoutDefaults []TimeoutConfig
	OptionsDefaults []OptionConfig
	Stats           *StatsInfo
	DNSResolvers    []*api.DNSResolver
	HTTPService     []*HTTPService
	TCPService      []*TCPService
	ErrorFiles      []*ErrorFile
	UserLists       []UserList
}

func (td TemplateData) UsesTLSAuth() bool {
	for _, srv := range td.HTTPService {
		if srv.TLSAuth != nil {
			return true
		}
	}
	for _, srv := range td.TCPService {
		if srv.TLSAuth != nil {
			return true
		}
	}
	return false
}

type TimeoutConfig struct {
	Phase    string
	Duration string
}

type OptionConfig struct {
	Option  string
	Enabled bool
}

type ErrorFile struct {
	StatusCode string
	Command    string
	Value      string
}

type SharedInfo struct {
	// Add accept-proxy to bind statements
	AcceptProxy           bool
	DefaultBackend        *Backend
	CORSConfig            CORSConfig
	ProxyBodySize         string
	EnableHSTS            bool
	HSTSMaxAge            int
	HSTSPreload           bool
	HSTSIncludeSubDomains bool
	WhitelistSourceRange  string
	MaxConnections        int
	UseNodePort           bool
	Limit                 *Limit
	HardStopAfter         string
}

type CORSConfig struct {
	CORSEnabled          bool
	CORSAllowedOrigin    string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
}

type StatsInfo struct {
	Port     int
	Username string
	PassWord string
}

type HTTPService struct {
	*SharedInfo

	FrontendName   string
	Address        string
	Port           int
	NodePort       int32
	NodePortFor443 int32
	OffloadSSL     bool
	ALPNOptions    string
	Proto          string
	FrontendRules  []string
	BasicAuth      *BasicAuth
	TLSAuth        *TLSAuth
	Hosts          []*HTTPHost
}

func (svc HTTPService) RedirectSSL() bool {
	for _, host := range svc.Hosts {
		for _, path := range host.Paths {
			if path.SSLRedirect {
				return true
			}
		}
	}
	return false
}

func (svc *HTTPService) RemoveBackendAuth() {
	for i := range svc.Hosts {
		for j := range svc.Hosts[i].Paths {
			if svc.Hosts[i].Paths[j].Backend != nil {
				svc.Hosts[i].Paths[j].Backend.BasicAuth = nil
			}
		}
	}
}

func (svc HTTPService) sortKey() string {
	if svc.OffloadSSL {
		return fmt.Sprintf("https://%d", svc.Port)
	}
	return fmt.Sprintf("http://%d", svc.Port)
}

type HTTPHost struct {
	Host         string
	Paths        []*HTTPPath
	ExternalAuth *ExternalAuth
}

type HTTPPath struct {
	//Host        string
	Path        string
	Backend     *Backend
	SSLRedirect bool
}

type TCPService struct {
	*SharedInfo

	FrontendName  string
	Address       string
	Port          string
	FrontendRules []string
	OffloadSSL    bool
	ALPNOptions   string
	Proto         string
	TLSAuth       *TLSAuth
	Hosts         []*TCPHost
}

func (svc TCPService) sortKey() string {
	return svc.Port
}

type TCPHost struct {
	Host    string
	Backend *Backend
}

type Backend struct {
	Name          string
	NameGenerated bool
	BasicAuth     *BasicAuth

	BackendRules []string
	// Deprecated
	RewriteRules []string
	// Deprecated
	HeaderRules []string
	Endpoints   []*Endpoint

	Sticky           bool
	StickyCookieName string
	StickyCookieHash string

	ALPNOptions string
	Proto       string

	LoadBalanceOn string
}

type ExternalAuth struct {
	AuthBackend     string
	AuthBackendPath string
	AuthPath        string
	SigninPath      string
	Paths           []string
}

func (be *Backend) canonicalize(hasDuplicate bool, address, host, port, path string) {
	if be.NameGenerated && hasDuplicate { // assign unique backend name
		hashed := md5.Sum([]byte(address + "-" + host + "-" + port + "-" + path))
		be.Name = be.Name + "-" + hex.EncodeToString(hashed[:])
	}
	sort.Slice(be.Endpoints, func(i, j int) bool { return be.Endpoints[i].IP < be.Endpoints[j].IP })
	if be.BasicAuth != nil {
		be.BasicAuth.canonicalize()
	}
}

type Endpoint struct {
	Name            string
	IP              string
	Port            string
	Weight          int
	MaxConnections  int
	ExternalName    string
	UseDNSResolver  bool
	DNSResolver     string
	CheckHealth     bool
	CheckHealthPort string
	TLSOption       string
	SendProxy       string
	AgentPort       string
	AgentInterval   string
}

type UserList struct {
	Name  string
	Users []AuthUser
}

func (ul *UserList) canonicalize() {
	sort.Slice(ul.Users, func(i, j int) bool { return ul.Users[i].Username < ul.Users[j].Username })
}

type BasicAuth struct {
	Realm     string
	UserLists []string
}

func (ba *BasicAuth) canonicalize() {
	sort.Strings(ba.UserLists)
}

type AuthUser struct {
	Username  string
	Password  string
	Encrypted bool
}

type Limit struct {
	Connection int
	TimeSecond int
	Rate       int
}

type TLSHeader struct {
	Header string
	Value  string
}

type TLSAuth struct {
	CAFile       string
	CRLFile      string
	VerifyClient string
	Headers      []TLSHeader
	ErrorPage    string
	ErrorPath    string
}
