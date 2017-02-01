package app

import (
	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/k8s-addons/pkg/events"
	"github.com/appscode/k8s-addons/pkg/stash"
	acw "github.com/appscode/k8s-addons/pkg/watcher"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/certificates"
	lbc "github.com/appscode/voyager/pkg/controller/ingress"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/util/wait"
)

type Watcher struct {
	acw.Watcher

	// name of the cloud provider
	ProviderName string

	// name of the cluster the daemon running.
	ClusterName string

	// Loadbalancer image name that will be used to create the LoadBalancer.
	LoadbalancerImage string

	IngressClass string
}

func (watch *Watcher) Run() {
	watch.setup()
	watch.Storage = &stash.Storage{}
	watch.Pod()
	watch.StatefulSet()
	watch.DaemonSet()
	watch.ReplicaSet()
	watch.Namespace()
	watch.Node()
	watch.Service()
	watch.RC()
	watch.Endpoint()

	watch.ExtendedIngress()
	watch.Ingress()
	watch.Alert()
	watch.Certificate()
}

func (w *Watcher) setup() {
	lbc.SetLoadbalancerImage(w.LoadbalancerImage)
	w.Watcher.Dispatch = w.Dispatch
}

func (w *Watcher) Dispatch(e *events.Event) error {
	if e.Ignorable() {
		return nil
	}
	log.Debugln("Dispatching event with resource", e.ResourceType, "event", e.EventType)

	if e.ResourceType == events.Ingress || e.ResourceType == events.ExtendedIngress {
		// Handle Ingress first
		err := lbc.NewEngressController(w.ClusterName,
			w.ProviderName,
			w.Client,
			w.AppsCodeExtensionClient,
			w.Storage, w.IngressClass).Handle(e)

		// Check the Ingress or Extended Ingress Annotations. To Work for auto certificate
		// operations.
		if err == nil {
			certController := certificates.NewController(w.Client, w.AppsCodeExtensionClient)
			certController.Handle(e)
		}
		return err
	}

	if e.ResourceType == events.Certificate {
		if e.EventType.IsAdded() || e.EventType.IsUpdated() {
			certController := certificates.NewController(w.Client, w.AppsCodeExtensionClient)
			certController.Handle(e)
		}
	}

	if e.ResourceType == events.Service {
		if e.EventType.IsAdded() || e.EventType.IsDeleted() {
			return lbc.UpgradeAllEngress(e.MetaData.Name+"."+e.MetaData.Namespace,
				w.ClusterName,
				w.ProviderName,
				w.Client,
				w.AppsCodeExtensionClient,
				w.Storage, w.IngressClass)
		}
	}

	if e.ResourceType == events.Endpoint {
		// Checking if this endpoint have a service or not. If
		// this do not have a Service we do not want to update our ingress
		_, err := w.Client.Core().Services(e.MetaData.Namespace).Get(e.MetaData.Name)
		if err == nil {
			log.Infoln("Endpoint has an service with name", e.MetaData.Name, e.MetaData.Namespace, "Event type", e.EventType.String())
			// Service exists. So we should process.
			if e.EventType.IsUpdated() {
				return lbc.UpgradeAllEngress(e.MetaData.Name+"."+e.MetaData.Namespace,
					w.ClusterName,
					w.ProviderName,
					w.Client,
					w.AppsCodeExtensionClient,
					w.Storage, w.IngressClass)
			}
		}
	}

	return nil
}

func (w *Watcher) Certificate() {
	log.Debugln("watching", events.Certificate.String())
	lw := &cache.ListWatch{
		ListFunc:  acw.CertificateListFunc(w.AppsCodeExtensionClient),
		WatchFunc: acw.CertificateWatchFunc(w.AppsCodeExtensionClient),
	}
	_, controller := w.Cache(events.Certificate, &aci.Certificate{}, lw)
	go controller.Run(wait.NeverStop)

	go certificates.NewCertificateSyncer(w.Client, w.AppsCodeExtensionClient).RunSync()
}
