package operator

import (
	"sync"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	kext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Operator struct {
	KubeClient      kubernetes.Interface
	CRDClient       kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient   cs.VoyagerV1beta1Interface
	PromClient      pcm.MonitoringV1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister
	Opt             config.Options

	recorder record.EventRecorder
	sync.Mutex
}

func New(
	kubeClient kubernetes.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1Interface,
	opt config.Options,
) *Operator {
	return &Operator{
		KubeClient:    kubeClient,
		CRDClient:     crdClient,
		VoyagerClient: extClient,
		PromClient:    promClient,
		Opt:           opt,
		recorder:      eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
}

func (op *Operator) Setup() error {
	log.Infoln("Ensuring CRD registration")

	crds := []*kext.CustomResourceDefinition{
		api.Ingress{}.CustomResourceDefinition(),
		api.Certificate{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(op.CRDClient, crds)
}

func (op *Operator) Run() {
	defer runtime.HandleCrash()

	informers := []cache.Controller{
		op.initNamespaceWatcher(),
		op.initNodeWatcher(),
		op.initConfigMapWatcher(),
		op.initDaemonSetWatcher(),
		op.initDeploymentWatcher(),
		op.initServiceWatcher(),
		op.initEndpointWatcher(),
		op.initIngresseWatcher(),
		op.initIngressCRDWatcher(),
		op.initCertificateCRDWatcher(),
		op.initSecretWatcher(),
	}
	if informer := op.initServiceMonitorWatcher(); informer != nil {
		informers = append(informers, informer)
	}
	for i := range informers {
		go informers[i].Run(wait.NeverStop)
	}
	go op.CheckCertificates()
}

func (op *Operator) listIngresses() ([]api.Ingress, error) {
	ing, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	eng, err := op.VoyagerClient.Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]api.Ingress, len(ing.Items))
	for i, item := range ing.Items {
		e, err := api.NewEngressFromIngress(item)
		if err != nil {
			continue
		}
		items[i] = *e
	}
	items = append(items, eng.Items...)
	return items, nil
}
