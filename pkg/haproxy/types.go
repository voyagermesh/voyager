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
}

type SharedInfo struct {
	// Add accept-proxy to bind statements
	AcceptProxy           bool
	DefaultBackend        *Backend
	Auth                  *AuthConfig
	EnableCORS            bool
	ProxyBodySize         string
	HSTSMaxAge            int
	HSTSPreload           bool
	HSTSIncludeSubDomains bool
}

type StatsInfo struct {
	Port     int
	Username string
	PassWord string
}

type HTTPService struct {
	*SharedInfo

	FrontendName  string
	Port          int
	NodePort      int32
	OffloadSSL    bool
	FrontendRules []string
	Paths         []*HTTPPath
}

func (svc HTTPService) sortKey() string {
	if svc.OffloadSSL {
		return fmt.Sprintf("https://%d", svc.Port)
	}
	return fmt.Sprintf("http://%d", svc.Port)
}

type HTTPPath struct {
	Host    string
	Path    string
	Backend Backend
}

func (svc HTTPPath) sortKey() string {
	return fmt.Sprintf("%s/%s", svc.Host, svc.Path)
}

type TCPService struct {
	*SharedInfo

	FrontendName  string
	Host          string
	Port          string
	FrontendRules []string
	SecretName    string
	PEMName       string
	Backend       Backend
	ALPNOptions   string
}

func (svc TCPService) sortKey() string {
	return fmt.Sprintf("%s:%s", svc.Host, svc.Port)
}

type Backend struct {
	Name string

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
}

type Endpoint struct {
	Name           string
	IP             string
	Port           string
	Weight         int
	ExternalName   string
	UseDNSResolver bool
	DNSResolver    string
	CheckHealth    bool

	TLSOption string
}

type AuthConfig struct {
	Realm string
	Users map[string][]AuthUser
}

type AuthUser struct {
	Username  string
	Password  string
	Encrypted bool
}
