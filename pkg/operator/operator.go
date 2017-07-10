package operator

import (
	"sync"
	"time"

	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/stash"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
)

type Options struct {
	CloudProvider      string
	CloudConfigFile    string
	HAProxyImage       string
	IngressClass       string
	ServiceAccountName string
}

type Operator struct {
	KubeClient clientset.Interface
	ExtClient  acs.ExtensionInterface
	PromClient pcm.MonitoringV1alpha1Interface
	Opt        Options

	recorder   record.EventRecorder
	SyncPeriod time.Duration
	Storage    stash.Storage
	sync.Mutex
}

func New(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt Options,
) *Operator {
	return &Operator{
		KubeClient: kubeClient,
		ExtClient:  extClient,
		PromClient: promClient,
		Opt:        opt,
		recorder:   eventer.NewEventRecorder(kubeClient, "Voyager operator"),
		SyncPeriod: 2 * time.Minute,
	}
}

func (c *Operator) Setup() error {
	log.Infoln("Ensuring TPR registration")

	if err := c.ensureThirdPartyResource("ingress" + "." + api.V1beta1SchemeGroupVersion.Group); err != nil {
		return err
	}
	if err := c.ensureThirdPartyResource("certificate" + "." + api.V1beta1SchemeGroupVersion.Group); err != nil {
		return err
	}
	return nil
}

func (c *Operator) ensureThirdPartyResource(resourceName string) error {
	_, err := c.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(resourceName, metav1.GetOptions{})
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
				Name: api.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	_, err = c.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Create(thirdPartyResource)
	return err
}

func (c *Operator) Run() {
	go c.WatchCertificateTPRs()
	go c.WatchConfigMaps()
	go c.WatchDaemonSets()
	go c.WatchDeployments()
	go c.WatchEndpoints()
	go c.WatchIngressTPRs()
	go c.WatchIngresses()
	go c.WatchNamespaces()
	go c.WatchServices()
}
