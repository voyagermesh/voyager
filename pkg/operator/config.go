package operator

import (
	hookapi "github.com/appscode/kutil/admission/api"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	voyagerinformers "github.com/appscode/voyager/client/informers/externalversions"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type OperatorConfig struct {
	config.Config

	ClientConfig   *rest.Config
	KubeClient     kubernetes.Interface
	CRDClient      kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient  cs.Interface
	PromClient     prom.MonitoringV1Interface
	AdmissionHooks []hookapi.AdmissionHook
}

func NewOperatorConfig(clientConfig *rest.Config) *OperatorConfig {
	return &OperatorConfig{
		ClientConfig: clientConfig,
	}
}

func (c *OperatorConfig) New() (*Operator, error) {
	op := &Operator{
		Config:                 c.Config,
		KubeClient:             c.KubeClient,
		kubeInformerFactory:    informers.NewFilteredSharedInformerFactory(c.KubeClient, c.ResyncPeriod, c.WatchNamespace, nil),
		CRDClient:              c.CRDClient,
		VoyagerClient:          c.VoyagerClient,
		voyagerInformerFactory: voyagerinformers.NewFilteredSharedInformerFactory(c.VoyagerClient, c.ResyncPeriod, c.WatchNamespace, nil),
		PromClient:             c.PromClient,
		recorder:               eventer.NewEventRecorder(c.KubeClient, "voyager operator"),
	}

	if err := op.ensureCustomResourceDefinitions(); err != nil {
		return nil, err
	}

	op.initIngressCRDWatcher()
	op.initIngressWatcher()
	op.initDeploymentWatcher()
	op.initServiceWatcher()
	op.initConfigMapWatcher()
	op.initEndpointWatcher()
	op.initSecretWatcher()
	op.initNodeWatcher()
	op.initServiceMonitorWatcher()
	op.initNamespaceWatcher()
	op.initCertificateCRDWatcher()

	return op, nil
}
