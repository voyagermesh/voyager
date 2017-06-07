package watcher

import (
	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/controller/certificates"
	"github.com/appscode/voyager/pkg/events"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/apps"
	ext "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/util/wait"
)

func (w *Watcher) Namespace() {
	log.Debugln("watching", events.Namespace.String())
	lw := &cache.ListWatch{
		ListFunc:  NamespaceListFunc(w.Client),
		WatchFunc: NamespaceWatchFunc(w.Client),
	}
	_, controller := w.Cache(events.Namespace, &kapi.Namespace{}, lw)
	go controller.Run(wait.NeverStop)
}

func (w *Watcher) Pod() {
	log.Debugln("watching", events.Pod.String())
	lw := &cache.ListWatch{
		ListFunc:  PodListFunc(w.Client),
		WatchFunc: PodWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.Pod, &kapi.Pod{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.PodStore = cache.StoreToPodLister{indexer}
}

func (w *Watcher) Service() {
	log.Debugln("watching", events.Service.String())
	lw := &cache.ListWatch{
		ListFunc:  ServiceListFunc(w.Client),
		WatchFunc: ServiceWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.Service, &kapi.Service{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.ServiceStore = cache.StoreToServiceLister{indexer}
}

func (w *Watcher) RC() {
	log.Debugln("watching", events.RC.String())
	lw := &cache.ListWatch{
		ListFunc:  ReplicationControllerListFunc(w.Client),
		WatchFunc: ReplicationControllerWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.RC, &kapi.ReplicationController{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.RcStore = cache.StoreToReplicationControllerLister{indexer}
}

func (w *Watcher) ReplicaSet() {
	log.Debugln("watching", events.ReplicaSet.String())
	lw := &cache.ListWatch{
		ListFunc:  ReplicaSetListFunc(w.Client),
		WatchFunc: ReplicaSetWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.ReplicaSet, &ext.ReplicaSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.ReplicaSetStore = cache.StoreToReplicaSetLister{indexer}
}

func (w *Watcher) StatefulSet() {
	log.Debugln("watching", events.StatefulSet.String())
	lw := &cache.ListWatch{
		ListFunc:  StatefulSetListFunc(w.Client),
		WatchFunc: StatefulSetWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.StatefulSet, &apps.StatefulSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.StatefulSetStore = cache.StoreToStatefulSetLister{indexer}
}

func (w *Watcher) DaemonSet() {
	log.Debugln("watching", events.DaemonSet.String())
	lw := &cache.ListWatch{
		ListFunc:  DaemonSetListFunc(w.Client),
		WatchFunc: DaemonSetWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.DaemonSet, &ext.DaemonSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.DaemonSetStore = cache.StoreToDaemonSetLister{indexer}
}

func (w *Watcher) Endpoint() {
	log.Debugln("watching", events.Endpoint.String())
	lw := &cache.ListWatch{
		ListFunc:  EndpointListFunc(w.Client),
		WatchFunc: EndpointWatchFunc(w.Client),
	}
	store, controller := w.CacheStore(events.Endpoint, &kapi.Endpoints{}, lw)
	go controller.Run(wait.NeverStop)
	w.Storage.EndpointStore = cache.StoreToEndpointsLister{store}
}

func (w *Watcher) Node() {
	log.Debugln("watching", events.Node.String())
	lw := &cache.ListWatch{
		ListFunc:  NodeListFunc(w.Client),
		WatchFunc: NodeWatchFunc(w.Client),
	}
	_, controller := w.CacheStore(events.Node, &kapi.Node{}, lw)
	go controller.Run(wait.NeverStop)
}

func (w *Watcher) Ingress() {
	log.Debugln("watching", events.Ingress.String())
	lw := &cache.ListWatch{
		ListFunc:  IngressListFunc(w.Client),
		WatchFunc: IngressWatchFunc(w.Client),
	}
	_, controller := w.Cache(events.Ingress, &ext.Ingress{}, lw)
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

	go certificates.NewCertificateSyncer(w.Client, w.ExtClient).RunSync()
}

func (w *Watcher) Deployment() {
	log.Debugln("watching", events.Deployments.String())
	lw := &cache.ListWatch{
		ListFunc:  DeploymentListFunc(w.Client),
		WatchFunc: DeploymentWatchFunc(w.Client),
	}
	indexer, controller := w.CacheIndexer(events.Deployments, &ext.Deployment{}, lw, nil)
	go controller.Run(wait.NeverStop)
	w.Storage.DeploymentStore = cache.StoreToDeploymentLister{indexer}
}

func (w *Watcher) ConfigMap() {
	log.Debugln("watching", events.ConfigMap.String())
	lw := &cache.ListWatch{
		ListFunc:  ConfigMapListFunc(w.Client),
		WatchFunc: ConfigMapWatchFunc(w.Client),
	}
	_, controller := w.CacheIndexer(events.ConfigMap, &kapi.ConfigMap{}, lw, nil)
	go controller.Run(wait.NeverStop)
}
