package operator

import (
	"github.com/appscode/go/log"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	voyagerinformers "github.com/appscode/voyager/client/informers/externalversions"
	api_listers "github.com/appscode/voyager/client/listers/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	kext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	core_listers "k8s.io/client-go/listers/core/v1"
	ext_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Operator struct {
	config.Config

	KubeClient     kubernetes.Interface
	WorkloadClient wcs.Interface
	CRDClient      kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient  cs.Interface
	PromClient     prom.MonitoringV1Interface

	kubeInformerFactory    informers.SharedInformerFactory
	voyagerInformerFactory voyagerinformers.SharedInformerFactory

	recorder record.EventRecorder

	// Certificate CRD
	crtQueue    *queue.Worker
	crtInformer cache.SharedIndexInformer
	crtLister   api_listers.CertificateLister

	// ConfigMap
	cfgQueue    *queue.Worker
	cfgInformer cache.SharedIndexInformer
	cfgLister   core_listers.ConfigMapLister

	// Deployment
	dpQueue    *queue.Worker
	dpInformer cache.SharedIndexInformer
	dpLister   apps_listers.DeploymentLister

	// DaemonSet
	dmQueue    *queue.Worker
	dmInformer cache.SharedIndexInformer
	dmLister   ext_listers.DaemonSetLister

	// Endpoint
	epQueue    *queue.Worker
	epInformer cache.SharedIndexInformer
	epLister   core_listers.EndpointsLister

	// Ingress CRD
	engQueue    *queue.Worker
	engInformer cache.SharedIndexInformer
	engLister   api_listers.IngressLister

	// Ingress
	ingQueue    *queue.Worker
	ingInformer cache.SharedIndexInformer
	ingLister   ext_listers.IngressLister

	// Namespace
	nsQueue    *queue.Worker
	nsInformer cache.SharedIndexInformer
	nsLister   core_listers.NamespaceLister

	// Node
	// nodeQueue    *queue.Worker
	nodeInformer cache.SharedIndexInformer
	nodeLister   core_listers.NodeLister

	// Secret
	secretQueue    *queue.Worker
	secretInformer cache.SharedIndexInformer
	secretLister   core_listers.SecretLister

	// Service Monitor
	smonQueue    *queue.Worker
	smonInformer cache.SharedIndexInformer
	// monLister   prom.ServiceMonitorLister

	// Service
	svcQueue    *queue.Worker
	svcInformer cache.SharedIndexInformer
	svcLister   core_listers.ServiceLister
}

func (op *Operator) ensureCustomResourceDefinitions() error {
	log.Infoln("Ensuring CRD registration")

	crds := []*kext.CustomResourceDefinition{
		api.Ingress{}.CustomResourceDefinition(),
		api.Certificate{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(op.CRDClient, crds)
}

func (op *Operator) RunInformers(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	go op.CheckCertificates()

	log.Infoln("Starting Voyager controller")
	op.kubeInformerFactory.Start(stopCh)
	op.voyagerInformerFactory.Start(stopCh)
	if op.smonInformer != nil {
		go op.smonInformer.Run(stopCh)
	}

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for t, v := range op.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			log.Fatalf("%v timed out waiting for caches to sync\n", t)
			return
		}
	}
	for t, v := range op.voyagerInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			log.Fatalf("%v timed out waiting for caches to sync\n", t)
			return
		}
	}
	if op.smonInformer != nil {
		if !cache.WaitForCacheSync(stopCh, op.smonInformer.HasSynced) {
			log.Fatalln("service monitor informer timed out waiting for caches to sync")
			return
		}
	}

	op.engQueue.Run(stopCh)
	op.ingQueue.Run(stopCh)
	op.dpQueue.Run(stopCh)
	op.svcQueue.Run(stopCh)
	op.cfgQueue.Run(stopCh)
	op.epQueue.Run(stopCh)
	op.secretQueue.Run(stopCh)
	op.nsQueue.Run(stopCh)
	op.crtQueue.Run(stopCh)
	if op.smonInformer != nil {
		op.smonQueue.Run(stopCh)
	}

	<-stopCh
	log.Infoln("Stopping Stash controller")
}

func (w *Operator) Run(stopCh <-chan struct{}) {
	// https://github.com/appscode/voyager/issues/346
	err := w.ValidateIngress()
	if err != nil {
		log.Errorln(err)
	}

	// https://github.com/appscode/voyager/pull/506
	err = w.MigrateCertificates()
	if err != nil {
		log.Errorln("Failed certificate migrations:", err)
	}
	// https://github.com/appscode/voyager/issues/229
	w.PurgeOffshootsWithDeprecatedLabels()
	// https://github.com/appscode/voyager/issues/446
	w.PurgeOffshootsDaemonSet()

	w.RunInformers(stopCh)
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
