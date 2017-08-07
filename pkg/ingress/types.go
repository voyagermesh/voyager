package ingress

import (
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
