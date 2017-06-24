package watcher

import (
	"reflect"
	"strings"
	"sync"
	"time"

	rt "github.com/appscode/go/runtime"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/certificates"
	"github.com/appscode/voyager/pkg/events"
	ingresscontroller "github.com/appscode/voyager/pkg/ingress"
	"github.com/appscode/voyager/pkg/stash"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	// name of the cloud provider
	ProviderName    string
	CloudConfigFile string

	// HAProxyImage is used to create HAProxy pods.
	HAProxyImage string

	IngressClass       string
	ServiceAccountName string

	KubeClient clientset.Interface
	ExtClient  acs.ExtensionInterface
	PromClient pcm.MonitoringV1alpha1Interface

	// sync time to sync the list.
	SyncPeriod time.Duration

	// lister store
	Storage stash.Storage

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
	ingresscontroller.SetLoadbalancerImage(w.HAProxyImage)
}

var resourceList []string = []string{
	"ingress",
	"certificate",
}

func (w *Watcher) ensureResource() {
	for _, resource := range resourceList {
		// This is version dependent
		_, err := w.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(resource+"."+api.V1beta1SchemeGroupVersion.Group, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			tpr := &extensions.ThirdPartyResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "extensions/v1beta1",
					Kind:       "ThirdPartyResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: resource + "." + api.V1beta1SchemeGroupVersion.Group,
				},
				Versions: []extensions.APIVersion{
					{
						Name: api.V1beta1SchemeGroupVersion.Version,
					},
				},
			}
			_, err := w.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
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
		err := ingresscontroller.NewEngressController(
			w.ProviderName,
			w.CloudConfigFile,
			w.KubeClient,
			w.ExtClient,
			w.PromClient,
			w.Storage,
			w.IngressClass,
			w.ServiceAccountName).Handle(e)

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
				w.ProviderName,
				w.CloudConfigFile,
				w.KubeClient,
				w.ExtClient,
				w.PromClient,
				w.Storage,
				w.IngressClass,
				w.ServiceAccountName)
		}
	case events.Endpoint:
		// Checking if this endpoint have a service or not. If
		// this do not have a Service we do not want to update our ingress
		_, err := w.KubeClient.CoreV1().Services(e.MetaData.Namespace).Get(e.MetaData.Name, metav1.GetOptions{})
		if err == nil {
			log.Infoln("Endpoint has an service with name", e.MetaData.Name, e.MetaData.Namespace, "Event type", e.EventType.String())
			// Service exists. So we should process.
			if e.EventType.IsUpdated() {
				return ingresscontroller.UpgradeAllEngress(
					e.MetaData.Name+"."+e.MetaData.Namespace,
					w.ProviderName,
					w.CloudConfigFile,
					w.KubeClient,
					w.ExtClient,
					w.PromClient,
					w.Storage,
					w.IngressClass,
					w.ServiceAccountName)
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
			sourceName, sourceNameFound := e.MetaData.Annotations[api.OriginName]
			sourceType, sourceTypeFound := e.MetaData.Annotations[api.OriginAPISchema]

			noAnnotationResource := false
			if !sourceNameFound && !sourceTypeFound {
				// Lets Check if those are old Ingress resource
				if strings.HasPrefix(e.MetaData.Name, api.VoyagerPrefix) {
					noAnnotationResource = true
					sourceName, sourceNameFound = e.MetaData.Name[len(api.VoyagerPrefix):], true
				}
			}

			if sourceNameFound {
				// deleted resource have source reference
				var ingressErr error
				var engress *api.Ingress
				if sourceType == api.APISchemaIngress {
					var resource *extensions.Ingress
					resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(e.MetaData.Namespace).Get(sourceName, metav1.GetOptions{})
					if ingressErr == nil {
						engress, _ = api.NewEngressFromIngress(resource)
					}
				} else if sourceType == api.APISchemaEngress {
					engress, ingressErr = w.ExtClient.Ingresses(e.MetaData.Namespace).Get(sourceName)
				} else if !sourceTypeFound {
					var resource *extensions.Ingress
					resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(e.MetaData.Namespace).Get(sourceName, metav1.GetOptions{})
					if ingressErr == nil {
						engress, _ = api.NewEngressFromIngress(resource)
					} else {
						engress, ingressErr = w.ExtClient.Ingresses(e.MetaData.Namespace).Get(sourceName)
					}
				}
				if ingressErr != nil {
					return
				}

				// Ingress Still exists, restore resource
				log.Infof("%s/%s required restore", e.EventType, e.ResourceType)
				obj, true := e.GetRuntimeObject()
				if true {
					var client rest.Interface
					switch e.ResourceType {
					case events.ConfigMap, events.Service:
						client = w.KubeClient.CoreV1().RESTClient()
					case events.Deployments, events.DaemonSet:
						client = w.KubeClient.ExtensionsV1beta1().RESTClient()
					}

					// Clean Default generated values
					metadata := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta")
					objectMeta, ok := metadata.Interface().(metav1.ObjectMeta)
					if ok {
						objectMeta.SetSelfLink("")
						objectMeta.SetResourceVersion("")

						if noAnnotationResource {
							// Old resource and annotations are missing so we need to add the annotations
							annotation := objectMeta.GetAnnotations()
							if annotation == nil {
								annotation = make(map[string]string)
							}
							annotation[api.OriginAPISchema] = engress.APISchema()
							annotation[api.OriginName] = sourceName

						}
						metadata.Set(reflect.ValueOf(objectMeta))
					}

					// Special treatments
					switch e.ResourceType {
					case events.Deployments:
						dp, ok := obj.(*extensions.Deployment)
						if ok {
							dp.Spec.Paused = false
							if types.Int32(dp.Spec.Replicas) < 1 {
								dp.Spec.Replicas = types.Int32P(1)
							}
						}
					case events.Service:
						svc, ok := obj.(*apiv1.Service)
						if ok {
							// Remove cluster IP
							svc.Spec.ClusterIP = ""
						}
						if svc.Name == engress.StatsServiceName() && !engress.Stats() {
							return
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

func (w *Watcher) Cache(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, cache.Controller) {
	var listWatch *cache.ListWatch
	if lw != nil {
		listWatch = lw
	} else {
		listWatch = cache.NewListWatchFromClient(w.KubeClient.CoreV1().RESTClient(), resource.String(), apiv1.NamespaceAll, fields.Everything())
	}

	return cache.NewInformer(
		listWatch,
		object,
		w.SyncPeriod,
		eventHandlerFuncs(w),
	)
}

func (w *Watcher) CacheStore(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(w.KubeClient.CoreV1().RESTClient(), resource.String(), apiv1.NamespaceAll, fields.Everything())
	}

	return stash.NewInformerPopulated(
		lw,
		object,
		w.SyncPeriod,
		eventHandlerFuncs(w),
	)
}

func (w *Watcher) CacheIndexer(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch, indexers cache.Indexers) (cache.Indexer, cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(w.KubeClient.CoreV1().RESTClient(), resource.String(), apiv1.NamespaceAll, fields.Everything())
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
