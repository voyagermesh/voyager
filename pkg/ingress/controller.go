package ingress

import (
	"sync"

	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
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

	// contains raw configMap data parsed from the cfg file.
	HAProxyConfig string

	// kubernetes client
	CloudManager cloudprovider.Interface
	sync.Mutex
}

func NewController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	services core.ServiceLister,
	endpoints core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) *Controller {
	ctrl := &Controller{
		KubeClient:      kubeClient,
		ExtClient:       extClient,
		PromClient:      promClient,
		ServiceLister:   services,
		EndpointsLister: endpoints,
		Opt:             opt,
		Ingress:         ingress,
		recorder:        eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		ctrl.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return ctrl
}

func (c *Controller) SupportsLBType() bool {
	switch c.Ingress.LBType() {
	case api.LBTypeLoadBalancer:
		return c.Opt.CloudProvider == "aws" ||
			c.Opt.CloudProvider == "gce" ||
			c.Opt.CloudProvider == "gke" ||
			c.Opt.CloudProvider == "azure" ||
			c.Opt.CloudProvider == "acs" ||
			c.Opt.CloudProvider == "minikube"
	case api.LBTypeNodePort, api.LBTypeHostPort:
		return c.Opt.CloudProvider != "acs"
	default:
		return false
	}
}
