/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	voyagerinformers "voyagermesh.dev/voyager/client/informers/externalversions"
	api_listers "voyagermesh.dev/voyager/client/listers/voyager/v1beta1"
	"voyagermesh.dev/voyager/pkg/config"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1"
	core_listers "k8s.io/client-go/listers/core/v1"
	ext_listers "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1beta1"
	"kmodules.xyz/client-go/apiextensions"
	"kmodules.xyz/client-go/tools/queue"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type Operator struct {
	config.Config

	ClientConfig   *rest.Config
	KubeClient     kubernetes.Interface
	WorkloadClient wcs.Interface
	CRDClient      crd_cs.Interface
	VoyagerClient  cs.Interface
	PromClient     prom.MonitoringV1Interface

	kubeInformerFactory    informers.SharedInformerFactory
	voyagerInformerFactory voyagerinformers.SharedInformerFactory

	recorder record.EventRecorder

	// ConfigMap
	cfgQueue    *queue.Worker
	cfgInformer cache.SharedIndexInformer
	cfgLister   core_listers.ConfigMapLister

	// Deployment
	dpQueue    *queue.Worker
	dpInformer cache.SharedIndexInformer
	dpLister   apps_listers.DeploymentLister

	// StatefulSet
	stsQueue    *queue.Worker
	stsInformer cache.SharedIndexInformer
	stsLister   apps_listers.StatefulSetLister

	// DaemonSet
	dmQueue    *queue.Worker
	dmInformer cache.SharedIndexInformer
	dmLister   apps_listers.DaemonSetLister

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
	klog.Infoln("Ensuring CRD registration")

	crds := []*apiextensions.CustomResourceDefinition{
		api.Ingress{}.CustomResourceDefinition(),
	}
	return apiextensions.RegisterCRDs(op.CRDClient, crds)
}

func (op *Operator) RunInformers(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	klog.Infoln("Starting Voyager controller")
	op.kubeInformerFactory.Start(stopCh)
	op.voyagerInformerFactory.Start(stopCh)
	if op.smonInformer != nil {
		go op.smonInformer.Run(stopCh)
	}

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for t, v := range op.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			klog.Fatalf("%v timed out waiting for caches to sync\n", t)
			return
		}
	}
	for t, v := range op.voyagerInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			klog.Fatalf("%v timed out waiting for caches to sync\n", t)
			return
		}
	}
	if op.smonInformer != nil {
		if !cache.WaitForCacheSync(stopCh, op.smonInformer.HasSynced) {
			klog.Fatalln("service monitor informer timed out waiting for caches to sync")
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
	if op.smonInformer != nil {
		op.smonQueue.Run(stopCh)
	}

	if op.EnableValidatingWebhook {
		cancel, _ := reg_util.SyncValidatingWebhookCABundle(op.ClientConfig, validatingWebhookConfig)
		defer cancel()
	}

	<-stopCh
	klog.Infoln("Stopping Voyager controller")
}

func (w *Operator) Run(stopCh <-chan struct{}) {
	// https://github.com/voyagermesh/voyager/issues/346
	err := w.ValidateIngress()
	if err != nil {
		klog.Errorln(err)
	}

	// https://github.com/voyagermesh/voyager/issues/229
	err = w.PurgeOffshootsWithDeprecatedLabels()
	if err != nil {
		klog.Errorln(err)
	}

	// https://github.com/voyagermesh/voyager/issues/446
	err = w.PurgeOffshootsDaemonSet()
	if err != nil {
		klog.Errorln(err)
	}

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
