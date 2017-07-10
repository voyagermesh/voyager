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
				if ingress, ok := obj.(*sapi.Ingress); ok {
					log.Infof("%s %s@%s added", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					go analytics.Send(ingress.GroupVersionKind().String(), "ADD", "success")
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldIngress, ok := old.(*sapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
					return
				}
				newIngress, ok := new.(*sapi.Ingress)
				if !ok {
					log.Errorln(errors.New("Invalid Ingress object"))
					return
				}
				fmt.Println(oldIngress.Name, newIngress.Name)
			},
			DeleteFunc: func(obj interface{}) {
				if ingress, ok := obj.(*sapi.Ingress); ok {
					log.Infof("%s %s@%s deleted", ingress.GroupVersionKind(), ingress.Name, ingress.Namespace)

					go analytics.Send(ingress.GroupVersionKind().String(), "DELETE", "success")
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
