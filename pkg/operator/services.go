package operator

import (
	"errors"
	"fmt"
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
func (c *Operator) WatchServices() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.CoreV1().Services(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.CoreV1().Services(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.Service{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s added", svc.Name, svc.Namespace)

				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldService, ok := old.(*apiv1.Service)
				if !ok {
					log.Errorln(errors.New("Invalid Service object"))
					return
				}
				newService, ok := new.(*apiv1.Service)
				if !ok {
					log.Errorln(errors.New("Invalid Service object"))
					return
				}

				fmt.Println(oldService.Name, newService.Name)

			},
			DeleteFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s deleted", svc.Name, svc.Namespace)

					c.restoreServiceIfRequired(svc)

				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (w *Operator) restoreServiceIfRequired(svc *apiv1.Service) error {
	if svc.Annotations == nil {
		return nil
	}

	sourceName, sourceNameFound := svc.Annotations[api.OriginName]
	sourceType, sourceTypeFound := svc.Annotations[api.OriginAPISchema]
	noAnnotationResource := false
	if !sourceNameFound && !sourceTypeFound {
		// Lets Check if those are old Ingress resource
		if strings.HasPrefix(svc.Name, api.VoyagerPrefix) {
			noAnnotationResource = true
			sourceName, sourceNameFound = svc.Name[len(api.VoyagerPrefix):], true
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
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(svc.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		}
	} else if sourceType == api.APISchemaEngress {
		engress, ingressErr = w.ExtClient.Ingresses(svc.Namespace).Get(sourceName)
	} else if !sourceTypeFound {
		var resource *extensions.Ingress
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(svc.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		} else {
			engress, ingressErr = w.ExtClient.Ingresses(svc.Namespace).Get(sourceName)
		}
	}
	if ingressErr != nil {
		return ingressErr
	}

	if svc.Name == engress.StatsServiceName() && !engress.Stats() {
		return nil
	}

	// Ingress Still exists, restore resource
	log.Infof("Service %s@%s requires restoration", svc.Name, svc.Namespace)
	svc.Spec.ClusterIP = "" // Remove cluster IP
	svc.SelfLink = ""
	svc.ResourceVersion = ""
	if noAnnotationResource {
		// Old resource and annotations are missing so we need to add the annotations
		if svc.Annotations == nil {
			svc.Annotations = make(map[string]string)
		}
		svc.Annotations[api.OriginAPISchema] = engress.APISchema()
		svc.Annotations[api.OriginName] = sourceName
	}

	_, err := w.KubeClient.CoreV1().Services(svc.Namespace).Create(svc)
	return err
}
