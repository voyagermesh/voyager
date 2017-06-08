package watcher

import (
	"reflect"
	"strings"
	"sync"
	"time"

	rt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/controller/certificates"
	ingresscontroller "github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/pkg/events"
	"github.com/appscode/voyager/pkg/stash"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
)

type Watcher struct {
	// name of the cloud provider
	ProviderName string

	// name of the cluster the daemon running.
	ClusterName string

	// HAProxyImage is used to create HAProxy pods.
	HAProxyImage string

	IngressClass string

	// kubernetes client to apiserver
	KubeClient clientset.Interface

	// client for getting the appscode extensions
	ExtClient acs.ExtensionInterface

	// sync time to sync the list.
	SyncPeriod time.Duration

	// lister store
	Storage *stash.Storage

	sync.Mutex
}

func (w *Watcher) Run() {
	defer rt.HandleCrash()
	w.setup()
	w.Pod()
	w.Service()
	w.Endpoint()

	// Watch for kubernetes resource, and restore resources that are created for
	// an ingress and the source ingress are not still deleted.
	w.Deployment()
	w.DaemonSet()
	w.ConfigMap()

	w.ExtendedIngress()
	w.Ingress()
	w.Certificate()
}

func (w *Watcher) setup() {
	w.ensureResource()
	w.Storage = &stash.Storage{}
	ingresscontroller.SetLoadbalancerImage(w.HAProxyImage)
}

var resourceList []string = []string{
	"ingress",
	"certificate",
}

func (w *Watcher) ensureResource() {
	for _, resource := range resourceList {
		// This is version dependent
		_, err := w.KubeClient.Extensions().ThirdPartyResources().Get(resource + "." + aci.V1beta1SchemeGroupVersion.Group)
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
			_, err := w.KubeClient.Extensions().ThirdPartyResources().Create(tpr)
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
			w.KubeClient,
			w.ExtClient,
			w.Storage, w.IngressClass).Handle(e)

		// Check the Ingress or Extended Ingress Annotations. To Work for auto certificate
		// operations.
		if err == nil {
			certController := certificates.NewController(w.KubeClient, w.ExtClient)
			certController.Handle(e)
		}
		sendAnalytics(e, err)
		return err
	case events.Certificate:
		var err error
		if e.EventType.IsAdded() || e.EventType.IsUpdated() {
			certController := certificates.NewController(w.KubeClient, w.ExtClient)
			err = certController.Handle(e)
		}
		sendAnalytics(e, err)
	case events.Service:
		if e.EventType.IsAdded() || e.EventType.IsDeleted() {
			w.restoreResourceIfRequired(e)
			return ingresscontroller.UpgradeAllEngress(
				e.MetaData.Name+"."+e.MetaData.Namespace,
				w.ClusterName,
				w.ProviderName,
				w.KubeClient,
				w.ExtClient,
				w.Storage, w.IngressClass)
		}
	case events.Endpoint:
		// Checking if this endpoint have a service or not. If
		// this do not have a Service we do not want to update our ingress
		_, err := w.KubeClient.Core().Services(e.MetaData.Namespace).Get(e.MetaData.Name)
		if err == nil {
			log.Infoln("Endpoint has an service with name", e.MetaData.Name, e.MetaData.Namespace, "Event type", e.EventType.String())
			// Service exists. So we should process.
			if e.EventType.IsUpdated() {
				return ingresscontroller.UpgradeAllEngress(
					e.MetaData.Name+"."+e.MetaData.Namespace,
					w.ClusterName,
					w.ProviderName,
					w.KubeClient,
					w.ExtClient,
					w.Storage, w.IngressClass)
			}
		}
	case events.ConfigMap, events.DaemonSet, events.Deployments:
		w.restoreResourceIfRequired(e)
	default:
		log.Infof("Event %s/%s is not handleable by voyager", e.EventType, e.ResourceType)
	}
	return nil
}

func (w *Watcher) restoreResourceIfRequired(e *events.Event) {
	log.Debugln("Dispatching event with resource", e.ResourceType, "event", e.EventType)
	switch e.ResourceType {
	case events.ConfigMap, events.DaemonSet, events.Deployments, events.Service:
		if e.EventType.IsDeleted() && e.MetaData.Annotations != nil {
			sourceName, sourceNameFound := e.MetaData.Annotations[ingresscontroller.LoadBalancerOriginName]
			sourceType, sourceTypeFound := e.MetaData.Annotations[ingresscontroller.LoadBalancerOriginAPIGroup]

			noAnnotationResource := false
			if !sourceNameFound && !sourceTypeFound {
				// Lets Check if those are old Ingress resource
				if strings.HasPrefix(e.MetaData.Name, ingresscontroller.VoyagerPrefix) {
					noAnnotationResource = true
					sourceName, sourceNameFound = e.MetaData.Name[len(ingresscontroller.VoyagerPrefix):], true
				}
			}

			if sourceNameFound {
				// deleted resource have source reference
				var ingressErr error
				var detectedAPIGroup string
				if sourceType == aci.APIGroupIngress {
					_, ingressErr = w.KubeClient.Extensions().Ingresses(e.MetaData.Namespace).Get(sourceName)
				} else if sourceType == aci.APIGroupEngress {
					_, ingressErr = w.ExtClient.Ingress(e.MetaData.Namespace).Get(sourceName)
				} else if !sourceTypeFound {
					_, ingressErr = w.KubeClient.Extensions().Ingresses(e.MetaData.Namespace).Get(sourceName)
					if ingressErr != nil {
						_, ingressErr = w.ExtClient.Ingress(e.MetaData.Namespace).Get(sourceName)
						if ingressErr == nil {
							detectedAPIGroup = aci.APIGroupEngress
						}
					} else {
						detectedAPIGroup = aci.APIGroupIngress
					}
				}

				if ingressErr != nil {
					return
				}

				// Ingress Still exists, restore resource
				log.Infof("%s/%s required restore", e.EventType, e.ResourceType)
				obj, true := e.GetRuntimeObject()
				if true {
					var client restclient.Interface
					switch e.ResourceType {
					case events.ConfigMap, events.Service:
						client = w.KubeClient.Core().RESTClient()
					case events.Deployments, events.DaemonSet:
						client = w.KubeClient.Extensions().RESTClient()
					}

					// Clean Default generated values
					metadata := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta")
					objectMeta, ok := metadata.Interface().(kapi.ObjectMeta)
					if ok {
						objectMeta.SetSelfLink("")
						objectMeta.SetResourceVersion("")

						if noAnnotationResource {
							// Old resource and annotations are missing so we need to add the annotations
							annotation := objectMeta.GetAnnotations()
							if annotation == nil {
								annotation = make(map[string]string)
							}
							annotation[ingresscontroller.LoadBalancerOriginAPIGroup] = detectedAPIGroup
							annotation[ingresscontroller.LoadBalancerOriginName] = sourceName

						}
						metadata.Set(reflect.ValueOf(objectMeta))
					}

					// Special treatment for Deployment
					if e.ResourceType == events.Deployments {
						dp, ok := obj.(*extensions.Deployment)
						if ok {
							dp.Spec.Paused = false
							if dp.Spec.Replicas < 1 {
								dp.Spec.Replicas = 1
							}
						}
					}

					resp, err := client.Post().
						Namespace(e.MetaData.Namespace).
						Resource(e.ResourceType.String()).
						Body(obj).
						Do().Raw()
					if err != nil {
						log.Errorln("Failed to create resource", e.ResourceType.String(), err)
						log.Errorln(string(resp))
					}
				}
			}
		}
	}
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

func (w *Watcher) Cache(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, *cache.Controller) {
	var listWatch *cache.ListWatch
	if lw != nil {
		listWatch = lw
	} else {
		listWatch = cache.NewListWatchFromClient(w.KubeClient.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}

	return cache.NewInformer(
		listWatch,
		object,
		w.SyncPeriod,
		eventHandlerFuncs(w),
	)
}

func (w *Watcher) CacheStore(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, *cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(w.KubeClient.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}

	return stash.NewInformerPopulated(
		lw,
		object,
		w.SyncPeriod,
		eventHandlerFuncs(w),
	)
}

func (w *Watcher) CacheIndexer(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch, indexers cache.Indexers) (cache.Indexer, *cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(w.KubeClient.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}
	if indexers == nil {
		indexers = cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	}

	return stash.NewIndexerInformerPopulated(
		lw,
		object,
		w.SyncPeriod,
		eventHandlerFuncs(w),
		indexers,
	)
}

func eventHandlerFuncs(k *Watcher) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			e := events.New(events.Added, obj)
			k.Dispatch(e)
		},
		DeleteFunc: func(obj interface{}) {
			e := events.New(events.Deleted, obj)
			k.Dispatch(e)
		},
		UpdateFunc: func(old, new interface{}) {
			if !reflect.DeepEqual(old, new) {
				e := events.New(events.Updated, old, new)
				k.Dispatch(e)
			}
		},
	}
}
