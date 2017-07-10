package operator

import (
	"errors"
	"fmt"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
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
				if pod, ok := obj.(*extensions.Ingress); ok {
					log.Infof("Ingress %s@%s added", pod.Name, pod.Namespace)

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

				fmt.Println(oldIngress.Name, newIngress.Name)

			},
			DeleteFunc: func(obj interface{}) {
				if pod, ok := obj.(*extensions.Ingress); ok {
					log.Infof("Ingress %s@%s deleted", pod.Name, pod.Namespace)

				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
