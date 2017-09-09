package operator

import (
	"sync"

	"github.com/appscode/log"
	tapi "github.com/appscode/voyager/apis/voyager"
	tapi_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	tcs "github.com/appscode/voyager/client/internalclientset/typed/voyager/internalversion"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/util"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Operator struct {
	KubeConfig      *rest.Config
	KubeClient      clientset.Interface
	CRDClient       apiextensionsclient.Interface
	ExtClient       tcs.VoyagerInterface
	PromClient      pcm.MonitoringV1alpha1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister
	Opt             config.Options

	recorder record.EventRecorder
	sync.Mutex
}

func New(
	config *rest.Config,
	kubeClient clientset.Interface,
	crdClient apiextensionsclient.Interface,
	extClient tcs.VoyagerInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt config.Options,
) *Operator {
	return &Operator{
		KubeConfig: config,
		KubeClient: kubeClient,
		CRDClient:  crdClient,
		ExtClient:  extClient,
		PromClient: promClient,
		Opt:        opt,
		recorder:   eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
}

func (op *Operator) Setup() error {
	log.Infoln("Ensuring TPR registration")

	if err := op.ensureCustomResourceDefinitions(); err != nil {
		return err
	}

	return nil
}

func (op *Operator) ensureCustomResourceDefinitions() error {
	crds := []*apiextensions.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   tapi.ResourceTypeIngress + "." + tapi_v1beta1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   tapi.GroupName,
				Version: tapi_v1beta1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   tapi.ResourceNameIngress,
					Plural:     tapi.ResourceTypeIngress,
					Kind:       tapi.ResourceKindIngress,
					ShortNames: []string{"ing"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   tapi.ResourceTypeCertificate + "." + tapi_v1beta1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   tapi.GroupName,
				Version: tapi_v1beta1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   tapi.ResourceNameCertificate,
					Plural:     tapi.ResourceTypeCertificate,
					Kind:       tapi.ResourceKindCertificate,
					ShortNames: []string{"cert"},
				},
			},
		},
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
	return util.WaitForCRDReady(
		op.KubeClient.CoreV1().RESTClient(),
		crds,
	)
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
		op.initIngressTPRWatcher(),
		op.initCertificateTPRWatcher(),
	}
	if informer := op.initServiceMonitorWatcher(); informer != nil {
		informers = append(informers, informer)
	}
	for i := range informers {
		go informers[i].Run(wait.NeverStop)
	}
	go certificate.CheckCertificates(op.KubeConfig, op.KubeClient, op.ExtClient, op.Opt)
}
