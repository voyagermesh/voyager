package ingress

import (
	"sync"

	api "github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/stash"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
)

type EngressController struct {
	ProviderName string
	IngressClass string

	// Engress object that created or updated.
	Resource *api.Ingress
	// kube options data
	SecretNames []string
	// contains raw configMap data parsed from the cfg file.
	ConfigData string

	// Ports contains a map of HAProxy port to Service Port (svc.TargetPort -> svc.Port).
	// HAProxy binds to the target ports. Service ports are used to open loadbalancer/firewall.
	// Usually target port == service port with one exception for LoadBalancer type service in AWS.
	// If AWS cert manager is used then a 80 -> 443 port mapping is added.
	Ports map[int]int
	// contains all the https host names.
	HostFilter []string
	// parsed ingress.
	Parsed HAProxyOptions

	// kubernetes client
	KubeClient   clientset.Interface
	ExtClient    acs.ExtensionInterface
	PromClient   pcm.MonitoringV1alpha1Interface
	CloudManager cloudprovider.Interface
	// endpoint cache store. contains all endpoints will be search with respect to services.
	Storage stash.Storage
	sync.Mutex
}

type HAProxyOptions struct {
	Timestamp int64
	// those options are get from annotations. applied globally
	// in all the sections.

	// stick requests to specified servers.
	Sticky  bool
	SSLCert bool

	// open up load balancer stats
	Stats bool
	// Basic auth to lb stats
	StatsUserName string
	StatsPassWord string
	StatsPort     int

	// Add accept-proxy to bind statements
	AcceptProxy bool

	DefaultBackend *Backend
	HttpsService   []*Service
	HttpService    []*Service
	TCPService     []*TCPService
	DNSResolvers   map[string]*api.DNSResolver
}

type Service struct {
	Name     string
	AclMatch string
	Host     string
	Backends *Backend
}

type TCPService struct {
	Name        string
	Host        string
	Port        string
	SecretName  string
	PEMName     string
	Backends    *Backend
	ALPNOptions string
}

type Backend struct {
	Name         string   `json:"Name,omitempty"`
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
}

// Loadbalancer image is an almost constant type.
// this will only be set at the runtime but only for once.
// once this is set the value can not be changed.
var loadbalancerImage string

func SetLoadbalancerImage(i string) {
	var once sync.Once
	once.Do(
		func() {
			loadbalancerImage = i
		},
	)
}

func GetLoadbalancerImage() string {
	return loadbalancerImage
}
