package app

import (
	"strings"

	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/controller/certificates"
	ingresscontroller "github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/pkg/events"
	"github.com/appscode/voyager/pkg/stash"
	acw "github.com/appscode/voyager/pkg/watcher"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
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
	watch.Pod()
	watch.Service()
	watch.Endpoint()

	watch.ExtendedIngress()
	watch.Ingress()
	watch.Certificate()
}

func (w *Watcher) setup() {
	w.ensureResource()
	w.Storage = &stash.Storage{}
	ingresscontroller.SetLoadbalancerImage(w.LoadbalancerImage)
	w.Watcher.Dispatch = w.Dispatch
}

var resourceList []string = []string{
	"ingress",
	"certificate",
}

func (w *Watcher) ensureResource() {
	for _, resource := range resourceList {
		// This is version dependent
		_, err := w.Client.Extensions().ThirdPartyResources().Get(resource + "." + aci.V1beta1SchemeGroupVersion.Group)
		if k8serrors.IsNotFound(err) {
			tpr := &extensions.ThirdPartyResource{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "extensions/v1beta1",
					Kind:       "ThirdPartyResource",
				},
				ObjectMeta: kapi.ObjectMeta{
					Name: resource + "." + aci.V1beta1SchemeGroupVersion.Group,
				},
				Versions: []extensions.APIVersion{
					{
						Name: aci.V1beta1SchemeGroupVersion.Version,
					},
				},
			}
			_, err := w.Client.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				// This should fail if there is one third party resource data missing.
				log.Fatalln(tpr.Name, "failed to create, causes", err.Error())
			}
		}
	}
}

func (w *Watcher) Dispatch(e *events.Event) error {
	if e.Ignorable() {
		return nil
	}
	log.Debugln("Dispatching event with resource", e.ResourceType, "event", e.EventType)
	switch e.ResourceType {
	case events.Ingress, events.ExtendedIngress:
		// Handle Ingress first
		err := ingresscontroller.NewEngressController(w.ClusterName,
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
		sendAnalytics(e, err)
		return err
	case events.Certificate:
		var err error
		if e.EventType.IsAdded() || e.EventType.IsUpdated() {
			certController := certificates.NewController(w.Client, w.AppsCodeExtensionClient)
			err = certController.Handle(e)
		}
		sendAnalytics(e, err)
	case events.Service:
		if e.EventType.IsAdded() || e.EventType.IsDeleted() {
			return ingresscontroller.UpgradeAllEngress(
				e.MetaData.Name+"."+e.MetaData.Namespace,
				w.ClusterName,
				w.ProviderName,
				w.Client,
				w.AppsCodeExtensionClient,
				w.Storage, w.IngressClass)
		}
	case events.Endpoint:
		// Checking if this endpoint have a service or not. If
		// this do not have a Service we do not want to update our ingress
		_, err := w.Client.Core().Services(e.MetaData.Namespace).Get(e.MetaData.Name)
		if err == nil {
			log.Infoln("Endpoint has an service with name", e.MetaData.Name, e.MetaData.Namespace, "Event type", e.EventType.String())
			// Service exists. So we should process.
			if e.EventType.IsUpdated() {
				return ingresscontroller.UpgradeAllEngress(
					e.MetaData.Name+"."+e.MetaData.Namespace,
					w.ClusterName,
					w.ProviderName,
					w.Client,
					w.AppsCodeExtensionClient,
					w.Storage, w.IngressClass)
			}
		}
	default:
		log.Infof("Event %s/%s is not handleable by voyager", e.EventType, e.ResourceType)
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

func sendAnalytics(e *events.Event, err error) {
	switch e.ResourceType {
	case events.Ingress, events.ExtendedIngress, events.Certificate:
		if e.EventType.IsAdded() || e.EventType.IsDeleted() {
			gv := ""
			runtimeObj, ok := e.RuntimeObj[0].(runtime.Object)
			if ok {
				gvk := runtimeObj.GetObjectKind().GroupVersionKind()
				if len(gvk.Group) > 0 && len(gvk.Version) > 0 {
					gv = strings.Join([]string{gvk.Group, gvk.Version}, "/") + "/"
				}
			}

			// Group Version is empty. Try Getting from SelfLink.
			if len(gv) <= 0 {
				// we extract group version from resource SelfLink. This guaranties
				// having GroupVersion.
				gv = e.MetaData.GetSelfLink()
				if len(gv) > 0 {
					tokens := strings.Split(strings.Trim(gv, "/"), "/")
					if len(tokens) >= 3 {
						gv = tokens[1] + "/" + tokens[2] + "/"
					}
				}
			}

			label := "success"
			if err != nil {
				label = "fail"
			}
			go analytics.Send(gv+e.ResourceType.String(), strings.ToLower(e.EventType.String()), label)
		}
	}
}
