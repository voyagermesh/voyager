package haproxy

import (
	"fmt"
	"sort"

	"github.com/appscode/voyager/api"
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
	AcceptProxy bool
	// stick requests to specified servers.
	Sticky         bool
	DefaultBackend *Backend
}

type StatsInfo struct {
	Port     int
	Username string
	PassWord string
}

type HTTPService struct {
	*SharedInfo

	FrontendName string
	Port         int
	NodePort     int32
	UsesSSL      bool
	Paths        []*HTTPPath
}

func (svc HTTPService) sortKey() string {
	if svc.UsesSSL {
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

	FrontendName string
	Host         string
	Port         string
	SecretName   string
	PEMName      string
	Backend      Backend
	ALPNOptions  string
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
