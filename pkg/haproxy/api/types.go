package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
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
	RedirectToPort int32
	OffloadSSL     bool
	FrontendRules  []string
	BasicAuth      *BasicAuth
	TLSAuth        *TLSAuth
	Hosts          []*HTTPHost
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
	Host  string
	Paths []*HTTPPath
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
	Host          string
	Port          string
	FrontendRules []string
	CertFile      string
	PEMName       string
	Backend       *Backend
	ALPNOptions   string
	TLSAuth       *TLSAuth
	SSLRedirect   bool
}

func (svc TCPService) sortKey() string {
	return fmt.Sprintf("%s:%s", svc.Host, svc.Port)
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
}

func (be *Backend) canonicalize(hasDuplicate bool, host, port, path string) {
	if be.NameGenerated && hasDuplicate { // assign unique backend name
		hashed := md5.Sum([]byte(host + "-" + port + "-" + path))
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
