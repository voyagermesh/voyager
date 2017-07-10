package operator

import (
	"errors"
	"fmt"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchIngresses() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.Ingress{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if ingress, ok := obj.(*extensions.Ingress); ok {
					log.Infof("%s %s@%s added", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					engress, err := api.NewEngressFromIngress(ingress)
					if err != nil {
						log.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", ingress.Name, ingress.Namespace, err)
						return
					}
					if !engress.ShouldHandleIngress(c.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)
						return
					}

					go analytics.Send(ingress.GroupVersionKind().String(), "ADD", "success")

				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldIngress, ok := old.(*extensions.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
					return
				}
				newIngress, ok := new.(*extensions.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
					return
				}

				oldEngress, err := api.NewEngressFromIngress(oldIngress)
				if err != nil {
					log.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", oldIngress.Name, oldIngress.Namespace, err)
					return
				}
				newEngress, err := api.NewEngressFromIngress(newIngress)
				if err != nil {
					log.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", newIngress.Name, newIngress.Namespace, err)
					return
				}

				if changed, _ := oldEngress.HasChanged(*newEngress); !changed {
					log.Infof("%s %s@%s has unchanged spec and annotations", newIngress.GroupVersionKind(), newIngress.Name, newIngress.Namespace)
					return
				}

				// check the case of switching ingress class

				fmt.Println(oldEngress.Name, newEngress.Name)
			},
			DeleteFunc: func(obj interface{}) {
				if ingress, ok := obj.(*extensions.Ingress); ok {
					log.Infof("%s %s@%s deleted", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					engress, err := api.NewEngressFromIngress(ingress)
					if err != nil {
						log.Errorf("Failed to convert Ingress %s@%s into Engress. Reason %v", ingress.Name, ingress.Namespace, err)
						return
					}
					if !engress.ShouldHandleIngress(c.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)
						return
					}

					go analytics.Send(ingress.GroupVersionKind().String(), "DELETE", "success")
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
