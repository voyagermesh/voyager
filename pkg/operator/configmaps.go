package operator

import (
	"strings"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchConfigMaps() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.CoreV1().ConfigMaps(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.CoreV1().ConfigMaps(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.ConfigMap{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if cfgmap, ok := obj.(*apiv1.ConfigMap); ok {
					log.Infof("ConfigMap %s@%s deleted", cfgmap.Name, cfgmap.Namespace)
					c.restoreConfigMapIfRequired(cfgmap)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (w *Operator) restoreConfigMapIfRequired(cfgmap *apiv1.ConfigMap) error {
	if cfgmap.Annotations == nil {
		return nil
	}

	sourceName, sourceNameFound := cfgmap.Annotations[api.OriginName]
	sourceType, sourceTypeFound := cfgmap.Annotations[api.OriginAPISchema]
	noAnnotationResource := false
	if !sourceNameFound && !sourceTypeFound {
		// Lets Check if those are old Ingress resource
		if strings.HasPrefix(cfgmap.Name, api.VoyagerPrefix) {
			noAnnotationResource = true
			sourceName, sourceNameFound = cfgmap.Name[len(api.VoyagerPrefix):], true
		}
	}
	if !sourceNameFound {
		return nil
	}

	// deleted resource have source reference
	var ingressErr error
	var engress *api.Ingress
	if sourceType == api.APISchemaIngress {
		var resource *extensions.Ingress
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(cfgmap.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		}
	} else if sourceType == api.APISchemaEngress {
		engress, ingressErr = w.ExtClient.Ingresses(cfgmap.Namespace).Get(sourceName)
	} else if !sourceTypeFound {
		var resource *extensions.Ingress
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(cfgmap.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		} else {
			engress, ingressErr = w.ExtClient.Ingresses(cfgmap.Namespace).Get(sourceName)
		}
	}
	if ingressErr != nil {
		return ingressErr
	}

	// Ingress Still exists, restore resource
	log.Infof("ConfigMap %s@%s requires restoration", cfgmap.Name, cfgmap.Namespace)
	cfgmap.SelfLink = ""
	cfgmap.ResourceVersion = ""
	if noAnnotationResource {
		// Old resource and annotations are missing so we need to add the annotations
		if cfgmap.Annotations == nil {
			cfgmap.Annotations = make(map[string]string)
		}
		cfgmap.Annotations[api.OriginAPISchema] = engress.APISchema()
		cfgmap.Annotations[api.OriginName] = sourceName
	}

	_, err := w.KubeClient.CoreV1().ConfigMaps(cfgmap.Namespace).Create(cfgmap)
	return err
}
