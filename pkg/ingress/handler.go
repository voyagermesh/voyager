package ingress

import (
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
)

func NewController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt config.Options,
	ingress *api.Ingress) *IngressController {
	h := &IngressController{
		KubeClient: kubeClient,
		ExtClient:  extClient,
		PromClient: promClient,
		Opt:        opt,
		Resource:   ingress,
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		h.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return h
}
