package operator

import (
	"errors"
	"fmt"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchIngressTPRs() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.ExtClient.Ingresses(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Ingress{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if engress, ok := obj.(*sapi.Ingress); ok {
					log.Infof("%s %s@%s added", engress.GroupVersionKind(), engress.Name, engress.Namespace)
					if !engress.ShouldHandleIngress(c.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", engress.GroupVersionKind(), engress.Name, engress.Namespace)
						return
					}
					go analytics.Send(engress.GroupVersionKind().String(), "ADD", "success")
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldEngress, ok := old.(*sapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
					return
				}
				newEngress, ok := new.(*sapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
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
				if engress, ok := obj.(*sapi.Ingress); ok {
					log.Infof("%s %s@%s deleted", engress.GroupVersionKind(), engress.Name, engress.Namespace)
					if !engress.ShouldHandleIngress(c.Opt.IngressClass) {
						log.Infof("%s %s@%s does not match ingress class", engress.GroupVersionKind(), engress.Name, engress.Namespace)
						return
					}
					go analytics.Send(engress.GroupVersionKind().String(), "DELETE", "success")
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
