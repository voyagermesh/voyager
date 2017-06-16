package watcher

import (
	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/certificates"
	"github.com/appscode/voyager/pkg/events"
apiv1 "k8s.io/client-go/pkg/api/v1"
extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
"k8s.io/client-go/tools/cache"
"k8s.io/apimachinery/pkg/util/wait"
)

func (w *Watcher) Pod() {
	log.Debugln("watching", events.Pod.String())
	lw := &cache.ListWatch{
		ListFunc:  PodListFunc(w.KubeClient),
		WatchFunc: PodWatchFunc(w.KubeClient),
	}
	indexer, controller := w.CacheIndexer(events.Pod, &apiv1.Pod{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.PodStore = cache.StoreToPodLister{indexer}
}

func (w *Watcher) Service() {
	log.Debugln("watching", events.Service.String())
	lw := &cache.ListWatch{
		ListFunc:  ServiceListFunc(w.KubeClient),
		WatchFunc: ServiceWatchFunc(w.KubeClient),
	}
	indexer, controller := w.CacheIndexer(events.Service, &apiv1.Service{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.ServiceStore = cache.StoreToServiceLister{indexer}
}

func (w *Watcher) Endpoint() {
	log.Debugln("watching", events.Endpoint.String())
	lw := &cache.ListWatch{
		ListFunc:  EndpointListFunc(w.KubeClient),
		WatchFunc: EndpointWatchFunc(w.KubeClient),
	}
	store, controller := w.CacheStore(events.Endpoint, &apiv1.Endpoints{}, lw)
	go controller.Run(wait.NeverStop)
	w.Storage.EndpointStore = cache.StoreToEndpointsLister{store}
}

func (w *Watcher) Deployment() {
	log.Debugln("watching", events.Deployments.String())
	lw := &cache.ListWatch{
		ListFunc:  DeploymentListFunc(w.KubeClient),
		WatchFunc: DeploymentWatchFunc(w.KubeClient),
	}
	indexer, controller := w.CacheIndexer(events.Deployments, &extensions.Deployment{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.DeploymentStore = cache.StoreToDeploymentLister{indexer}
}

func (w *Watcher) DaemonSet() {
	log.Debugln("watching", events.DaemonSet.String())
	lw := &cache.ListWatch{
		ListFunc:  DaemonSetListFunc(w.KubeClient),
		WatchFunc: DaemonSetWatchFunc(w.KubeClient),
	}
	indexer, controller := w.CacheIndexer(events.DaemonSet, &extensions.DaemonSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.DaemonSetStore = cache.StoreToDaemonSetLister{indexer}
}

func (w *Watcher) ConfigMap() {
	log.Debugln("watching", events.ConfigMap.String())
	lw := &cache.ListWatch{
		ListFunc:  ConfigMapListFunc(w.KubeClient),
		WatchFunc: ConfigMapWatchFunc(w.KubeClient),
	}
	_, controller := w.CacheIndexer(events.ConfigMap, &apiv1.ConfigMap{}, lw, nil)
	go controller.Run(wait.NeverStop)
}

func (w *Watcher) Ingress() {
	log.Debugln("watching", events.Ingress.String())
	lw := &cache.ListWatch{
		ListFunc:  IngressListFunc(w.KubeClient),
		WatchFunc: IngressWatchFunc(w.KubeClient),
	}
	_, controller := w.Cache(events.Ingress, &extensions.Ingress{}, lw)
	go controller.Run(wait.NeverStop)
}

func (w *Watcher) ExtendedIngress() {
	log.Debugln("watching", events.ExtendedIngress.String())
	lw := &cache.ListWatch{
		ListFunc:  ExtendedIngressListFunc(w.ExtClient),
		WatchFunc: ExtendedIngressWatchFunc(w.ExtClient),
	}
	_, controller := w.Cache(events.ExtendedIngress, &aci.Ingress{}, lw)
	go controller.Run(wait.NeverStop)
}

func (w *Watcher) Certificate() {
	log.Debugln("watching", events.Certificate.String())
	lw := &cache.ListWatch{
		ListFunc:  CertificateListFunc(w.ExtClient),
		WatchFunc: CertificateWatchFunc(w.ExtClient),
	}
	_, controller := w.Cache(events.Certificate, &aci.Certificate{}, lw)
	go controller.Run(wait.NeverStop)

	go certificates.NewCertificateSyncer(w.KubeClient, w.ExtClient).RunSync()
}
