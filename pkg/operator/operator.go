package operator

import (
	"fmt"
	"sync"
	"time"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	api_listers "github.com/appscode/voyager/listers/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	kext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	core_listers "k8s.io/client-go/listers/core/v1"
	ext_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Operator struct {
	KubeClient    kubernetes.Interface
	CRDClient     kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient cs.VoyagerV1beta1Interface
	PromClient    pcm.MonitoringV1Interface
	Opt           config.Options

	recorder record.EventRecorder
	sync.Mutex

	// Certificate CRD
	certQueue    workqueue.RateLimitingInterface
	certIndexer  cache.Indexer
	certInformer cache.Controller
	certLister   api_listers.CertificateLister

	// ConfigMap
	cfgQueue    workqueue.RateLimitingInterface
	cfgIndexer  cache.Indexer
	cfgInformer cache.Controller
	cfgLister   core_listers.ConfigMapLister

	// Deployment
	dpQueue    workqueue.RateLimitingInterface
	dpIndexer  cache.Indexer
	dpInformer cache.Controller
	dpLister   apps_listers.DeploymentLister

	// Endpoint
	epQueue    workqueue.RateLimitingInterface
	epIndexer  cache.Indexer
	epInformer cache.Controller
	epLister   core_listers.EndpointsLister

	// Ingress CRD
	engQueue    workqueue.RateLimitingInterface
	engIndexer  cache.Indexer
	engInformer cache.Controller
	engLister   api_listers.IngressLister

	// Ingress
	ingQueue    workqueue.RateLimitingInterface
	ingIndexer  cache.Indexer
	ingInformer cache.Controller
	ingLister   ext_listers.IngressLister

	// Namespace
	nsQueue    workqueue.RateLimitingInterface
	nsIndexer  cache.Indexer
	nsInformer cache.Controller
	nsLister   core_listers.NamespaceLister

	// Node
	// nodeQueue    workqueue.RateLimitingInterface
	nodeIndexer  cache.Indexer
	nodeInformer cache.Controller
	nodeLister   core_listers.NodeLister

	// Secret
	secretQueue    workqueue.RateLimitingInterface
	secretIndexer  cache.Indexer
	secretInformer cache.Controller
	secretLister   core_listers.SecretLister

	// Service Monitor
	monQueue    workqueue.RateLimitingInterface
	monIndexer  cache.Indexer
	monInformer cache.Controller
	// monLister   prom.ServiceMonitorLister

	// Service
	svcQueue    workqueue.RateLimitingInterface
	svcIndexer  cache.Indexer
	svcInformer cache.Controller
	svcLister   core_listers.ServiceLister
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
	if err := op.ensureCustomResourceDefinitions(); err != nil {
		return err
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

	return nil
}

func (op *Operator) ensureCustomResourceDefinitions() error {
	log.Infoln("Ensuring CRD registration")

	crds := []*kext.CustomResourceDefinition{
		api.Ingress{}.CustomResourceDefinition(),
		api.Certificate{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(op.CRDClient, crds)
}

func (op *Operator) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	go op.CheckCertificates()

	defer op.engQueue.ShutDown()
	defer op.ingQueue.ShutDown()
	defer op.dpQueue.ShutDown()
	defer op.svcQueue.ShutDown()
	defer op.cfgQueue.ShutDown()
	defer op.epQueue.ShutDown()
	defer op.secretQueue.ShutDown()
	defer op.nsQueue.ShutDown()
	defer op.certQueue.ShutDown()

	defer func() {
		if op.monInformer != nil {
			op.monQueue.ShutDown()
		}
	}()

	log.Infoln("Starting Voyager controller")

	go op.engInformer.Run(stopCh)
	go op.ingInformer.Run(stopCh)
	go op.dpInformer.Run(stopCh)
	go op.svcInformer.Run(stopCh)
	go op.cfgInformer.Run(stopCh)
	go op.epInformer.Run(stopCh)
	go op.secretInformer.Run(stopCh)
	go op.nodeInformer.Run(stopCh)
	go op.nsInformer.Run(stopCh)
	go op.certInformer.Run(stopCh)

	if op.monInformer != nil {
		op.monInformer.Run(stopCh)
	}

	if !cache.WaitForCacheSync(stopCh, op.engInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.ingInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.dpInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.svcInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.cfgInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.epInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.secretInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.nsInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, op.certInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if op.monInformer != nil {
		if !cache.WaitForCacheSync(stopCh, op.monInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(op.runEngressWatcher, time.Second, stopCh)
		go wait.Until(op.runIngressWatcher, time.Second, stopCh)
		go wait.Until(op.runDeploymentWatcher, time.Second, stopCh)
		go wait.Until(op.runServiceWatcher, time.Second, stopCh)
		go wait.Until(op.runConfigMapWatcher, time.Second, stopCh)
		go wait.Until(op.runEndpointWatcher, time.Second, stopCh)
		go wait.Until(op.runSecretWatcher, time.Second, stopCh)
		go wait.Until(op.runNamespaceWatcher, time.Second, stopCh)
		go wait.Until(op.runCertificateWatcher, time.Second, stopCh)

		if op.monInformer != nil {
			go wait.Until(op.runServiceMonitorWatcher, time.Second, stopCh)
		}
	}

	<-stopCh
	log.Infoln("Stopping Stash controller")
}

func (op *Operator) listIngresses() ([]api.Ingress, error) {
	ingList, err := op.ingLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	engList, err := op.engLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	items := make([]api.Ingress, len(engList))
	for i, item := range engList {
		items[i] = *item
	}
	for _, item := range ingList {
		if e, err := api.NewEngressFromIngress(item); err == nil {
			items = append(items, *e)
		}
	}
	return items, nil
}
