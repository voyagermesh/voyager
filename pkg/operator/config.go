package operator

import (
	"time"

	cs "github.com/appscode/voyager/client"
	voyagerinformers "github.com/appscode/voyager/informers/externalversions"
	hookapi "github.com/appscode/voyager/pkg/admission/api"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type OperatorConfig struct {
	ClientConfig *rest.Config

	OpsAddress     string
	WatchNamespace string

	KubeClient    kubernetes.Interface
	CRDClient     kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient cs.Interface
	PromClient    prom.MonitoringV1Interface
	options       config.OperatorOptions

	AdmissionHooks []hookapi.AdmissionHook
}

func NewOperatorConfig(clientConfig *rest.Config) *OperatorConfig {
	return &OperatorConfig{
		ClientConfig: clientConfig,
	}
}

func (c *OperatorConfig) New() (*Operator, error) {
	op := &Operator{
		KubeClient:             c.KubeClient,
		kubeInformerFactory:    informers.NewFilteredSharedInformerFactory(c.KubeClient, 10*time.Minute, c.WatchNamespace, nil),
		CRDClient:              c.CRDClient,
		VoyagerClient:          c.VoyagerClient,
		voyagerInformerFactory: voyagerinformers.NewFilteredSharedInformerFactory(c.VoyagerClient, 10*time.Minute, c.WatchNamespace, nil),
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
