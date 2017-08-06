package ingress

import (
	"fmt"
	"sync"

	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
)

type Controller struct {
	KubeClient      clientset.Interface
	ExtClient       acs.ExtensionInterface
	PromClient      pcm.MonitoringV1alpha1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister

	recorder record.EventRecorder

	Opt config.Options

	// Engress object that created or updated.
	Ingress *api.Ingress

	// Ports contains a map of Service Port to HAProxy port (svc.Port -> svc.TargetPort).
	// HAProxy pods binds to the target ports. Service ports are used to open loadbalancer/firewall.
	// Usually target port == service port with one exception for LoadBalancer type service in AWS.
	// If AWS cert manager is used then a 443 -> 80 port mapping is added.
	PortMapping map[int]Target
	// parsed ingress.
	TemplateData IngressInfo
	// contains raw configMap data parsed from the cfg file.
	HAProxyConfig string

	// kubernetes client
	CloudManager cloudprovider.Interface
	sync.Mutex
}

type Target struct {
	PodPort  int
	NodePort int
}

type IngressInfo struct {
	Timestamp int64
	// those options are get from annotations. applied globally
	// in all the sections.

	// stick requests to specified servers.
	Sticky  bool
	SSLCert bool

	TimeoutDefaults map[string]string

	// open up load balancer stats
	Stats bool
	// Basic auth to lb stats
	StatsUserName string
	StatsPassWord string
	StatsPort     int

	// Add accept-proxy to bind statements
	AcceptProxy    bool
	DefaultBackend *Backend
	HTTPService    []*HTTPService
	TCPService     []*TCPService
	DNSResolvers   map[string]*api.DNSResolver
}

type HTTPService struct {
	Name    string
	Port    int
	UsesSSL bool
	Paths   []*HTTPPath
}

func (svc HTTPService) SortKey() string {
	if svc.UsesSSL {
		return fmt.Sprintf("https://%d", svc.Port)
	}
	return fmt.Sprintf("http://%d", svc.Port)
}

type HTTPPath struct {
	Name    string
	Host    string
	Path    string
	Backend Backend
}

func (svc HTTPPath) SortKey() string {
	return fmt.Sprintf("%s/%s", svc.Host, svc.Path)
}

type TCPService struct {
	Name        string
	Host        string
	Port        string
	UsesSSL     bool
	PEMName     string
	Backend     Backend
	ALPNOptions string
}

func (svc TCPService) SortKey() string {
	return fmt.Sprintf("%s:%s", svc.Host, svc.Port)
}

type Backend struct {
	BackendRules []string `json:"BackendRules,omitempty"`
	// Deprecated
	RewriteRules []string `json:"RewriteRules,omitempty"`
	// Deprecated
	HeaderRules []string    `json:"HeaderRules,omitempty"`
	Endpoints   []*Endpoint `json:"Endpoints,omitempty"`
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
}
