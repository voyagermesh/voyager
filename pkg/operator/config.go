package operator

import (
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	reg_util "github.com/appscode/kutil/admissionregistration/v1beta1"
	"github.com/appscode/kutil/discovery"
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

const (
	validatingWebhook = "admission.voyager.appscode.com"
)

type OperatorConfig struct {
	config.Config

	ClientConfig   *rest.Config
	KubeClient     kubernetes.Interface
	WorkloadClient wcs.Interface
	CRDClient      kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient  cs.Interface
	PromClient     prom.MonitoringV1Interface
	AdmissionHooks []hooks.AdmissionHook
}

func NewOperatorConfig(clientConfig *rest.Config) *OperatorConfig {
	return &OperatorConfig{
		ClientConfig: clientConfig,
	}
}

func (c *OperatorConfig) New() (*Operator, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	op := &Operator{
		Config:                 c.Config,
		ClientConfig:           c.ClientConfig,
		KubeClient:             c.KubeClient,
		WorkloadClient:         c.WorkloadClient,
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
	if err := reg_util.UpdateValidatingWebhookCABundle(c.ClientConfig, validatingWebhook); err != nil {
		return nil, err
	}

	op.initIngressCRDWatcher()
	op.initIngressWatcher()
	op.initDeploymentWatcher()
	op.initDaemonSetWatcher()
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
