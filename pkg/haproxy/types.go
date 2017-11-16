package haproxy

import (
	"fmt"
	"sort"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
)

type TemplateData struct {
	*SharedInfo
	TimeoutDefaults map[string]string
	OptionsDefaults map[string]bool
	Stats           *StatsInfo
	DNSResolvers    []*api.DNSResolver
	HTTPService     []*HTTPService
	TCPService      []*TCPService
	ErrorFiles      []*ErrorFile
	UserLists       []UserList
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
	EnableCORS            bool
	ProxyBodySize         string
	EnableHSTS            bool
	HSTSMaxAge            int
	HSTSPreload           bool
	HSTSIncludeSubDomains bool
	WhitelistSourceRange  string
	MaxConnections        int
	ForceMatchServicePort bool
	Limit                 *Limit
}

type StatsInfo struct {
	Port     int
	Username string
	PassWord string
}

type HTTPService struct {
	*SharedInfo

	FrontendName  string
	Address       string
	Port          int
	NodePort      int32
	OffloadSSL    bool
	FrontendRules []string
	BasicAuth     *BasicAuth
	TLSAuth       *TLSAuth
	Paths         []*HTTPPath
}

func (svc *HTTPService) RemoveBackendAuth() {
	for i := range svc.Paths {
		svc.Paths[i].Backend.BasicAuth = nil
	}
}

func (svc HTTPService) sortKey() string {
	if svc.OffloadSSL {
		return fmt.Sprintf("https://%d", svc.Port)
	}
	return fmt.Sprintf("http://%d", svc.Port)
}

type HTTPPath struct {
	Host        string
	Path        string
	Backend     Backend
	SSLRedirect bool
}

func (svc HTTPPath) sortKey() string {
	return fmt.Sprintf("%s/%s", svc.Host, svc.Path)
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
	Backend       Backend
	ALPNOptions   string
	TLSAuth       *TLSAuth
	SSLRedirect   bool
}

func (svc TCPService) sortKey() string {
	return fmt.Sprintf("%s:%s", svc.Host, svc.Port)
}

type Backend struct {
	Name      string
	BasicAuth *BasicAuth

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

func (be *Backend) canonicalize() {
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

type TLSAuth struct {
	CAFile       string
	CRLFile      string
	VerifyClient string
	Headers      map[string]string
	ErrorPage    string
	ErrorPath    string
}
