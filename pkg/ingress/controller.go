package ingress

import (
	"sync"

	api "github.com/appscode/voyager/apis/voyager"
	acs "github.com/appscode/voyager/client/internalclientset/typed/voyager/internalversion"
	"github.com/appscode/voyager/pkg/config"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
)

type Controller interface {
	IsExists() bool
	Create() error
	Update(mode UpdateMode, old *api.Ingress) error
	Delete()
	EnsureFirewall(svc *apiv1.Service) error
}

type controller struct {
	KubeClient      clientset.Interface
	CRDClient       apiextensionsclient.Interface
	ExtClient       acs.VoyagerInterface
	PromClient      pcm.MonitoringV1alpha1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister

	recorder record.EventRecorder

	Opt config.Options

	// Engress object that created or updated.
	Ingress *api.Ingress

	// contains raw configMap data parsed from the cfg file.
	HAProxyConfig string

	sync.Mutex
}

func NewController(
	kubeClient clientset.Interface,
	crdClient apiextensionsclient.Interface,
	extClient acs.VoyagerInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	serviceLister core.ServiceLister,
	endpointsLister core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	switch ingress.LBType() {
	case api.LBTypeHostPort:
		return NewHostPortController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeNodePort:
		return NewNodePortController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeLoadBalancer:
		return NewLoadBalancerController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	}
	return nil
}
