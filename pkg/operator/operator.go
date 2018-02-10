package operator

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/pat"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client"
	voyagerinformers "github.com/appscode/voyager/informers/externalversions"
	api_listers "github.com/appscode/voyager/listers/voyager/v1beta1"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	MaxNumRequeues int
	NumThreads     int
	IngressClass   string
	WatchNamespace string
	OpsAddress     string

	CloudProvider               string
	OperatorNamespace           string
	OperatorService             string
	EnableRBAC                  bool
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string
	QPS                         float32
	Burst                       int
	RestrictToOperatorNamespace bool
	CloudConfigFile             string

	KubeClient    kubernetes.Interface
	CRDClient     kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient cs.Interface
	PromClient    prom.MonitoringV1Interface
	// options       config.OperatorOptions

	kubeInformerFactory    informers.SharedInformerFactory
	voyagerInformerFactory voyagerinformers.SharedInformerFactory

	recorder record.EventRecorder
	sync.Mutex

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
		op.smonInformer.Run(stopCh)
	}

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, v := range op.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	for _, v := range op.voyagerInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	if op.smonInformer != nil {
		if !cache.WaitForCacheSync(stopCh, op.smonInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
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
		log.Fatalln("Failed certificate migrations:", err)
	}
	// https://github.com/appscode/voyager/issues/229
	w.PurgeOffshootsWithDeprecatedLabels()
	// https://github.com/appscode/voyager/issues/446
	w.PurgeOffshootsDaemonSet()

	stop := make(chan struct{})
	defer close(stop)
	go w.RunInformers(stop)

	m := pat.New()
	m.Get("/metrics", promhttp.Handler())
	http.Handle("/", m)
	log.Infoln("Listening on", w.OpsAddress)
	log.Fatal(http.ListenAndServe(w.OpsAddress, nil))
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
