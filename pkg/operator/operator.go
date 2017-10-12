package operator

import (
	"sync"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	tcs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
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
	CRDClient       apiextensionsclient.Interface
	VoyagerClient   tcs.VoyagerV1beta1Interface
	PromClient      pcm.MonitoringV1alpha1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister
	Opt             config.Options

	recorder record.EventRecorder
	sync.Mutex
}

func New(
	kubeClient kubernetes.Interface,
	crdClient apiextensionsclient.Interface,
	extClient tcs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1alpha1Interface,
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

	if err := op.ensureCustomResourceDefinitions(); err != nil {
		return err
	}

	return nil
}

func (op *Operator) ensureCustomResourceDefinitions() error {
	crds := []*apiextensions.CustomResourceDefinition{
		api.Ingress{}.CustomResourceDefinition(),
		api.Certificate{}.CustomResourceDefinition(),
	}
	for _, crd := range crds {
		_, err := op.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = op.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return kutil.WaitForCRDReady(op.KubeClient.CoreV1().RESTClient(), crds)
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
