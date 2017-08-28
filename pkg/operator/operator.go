package operator

import (
	"sync"

	"github.com/appscode/log"
	tapi "github.com/appscode/voyager/api"
	tcs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Operator struct {
	KubeConfig      *rest.Config
	KubeClient      clientset.Interface
	ExtClient       tcs.ExtensionInterface
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
	extClient tcs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt config.Options,
) *Operator {
	return &Operator{
		KubeConfig: config,
		KubeClient: kubeClient,
		ExtClient:  extClient,
		PromClient: promClient,
		Opt:        opt,
		recorder:   eventer.NewEventRecorder(kubeClient, "voyager operator"),
	}
}

func (op *Operator) Setup() error {
	log.Infoln("Ensuring TPR registration")

	if err := op.ensureThirdPartyResource(tapi.ResourceNameIngress + "." + tapi.V1beta1SchemeGroupVersion.Group); err != nil {
		return err
	}
	if err := op.ensureThirdPartyResource(tapi.ResourceNameCertificate + "." + tapi.V1beta1SchemeGroupVersion.Group); err != nil {
		return err
	}
	return nil
}

func (op *Operator) ensureThirdPartyResource(resourceName string) error {
	_, err := op.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(resourceName, metav1.GetOptions{})
	if !kerr.IsNotFound(err) {
		return err
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
			Labels: map[string]string{
				"app": "voyager",
			},
		},
		Description: "Voyager by AppsCode - Secure Ingress Controller for Kubernetes",
		Versions: []extensions.APIVersion{
			{
				Name: tapi.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	_, err = op.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Create(thirdPartyResource)
	return err
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
