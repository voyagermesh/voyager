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
	Parsed TemplateInfo
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

type TemplateInfo struct {
	SharedInfo
	TimeoutDefaults map[string]string
	Stats           *StatsInfo
	DNSResolvers    map[string]*api.DNSResolver
	DefaultBackend  *Backend
	HTTPService     []*HTTPService
	TCPService      []*TCPService
}

type SharedInfo struct {
	// Add accept-proxy to bind statements
	AcceptProxy bool
	// stick requests to specified servers.
	Sticky bool
}

type StatsInfo struct {
	Port     int
	UserName string
	PassWord string
}

type HTTPService struct {
	SharedInfo

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
	SharedInfo

	Name        string
	Host        string
	Port        string
	SecretName  string
	PEMName     string
	Backend     Backend
	ALPNOptions string
}

func (svc TCPService) SortKey() string {
	return fmt.Sprintf("%s:%s", svc.Host, svc.Port)
}

type Backend struct {
	BackendRules []string
	// Deprecated
	RewriteRules []string
	// Deprecated
	HeaderRules []string
	Endpoints   []*Endpoint
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
