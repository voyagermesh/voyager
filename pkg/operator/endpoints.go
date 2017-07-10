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
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchEndpoints() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.CoreV1().Endpoints(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.CoreV1().Endpoints(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.Endpoints{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if pod, ok := obj.(*apiv1.Endpoints); ok {
					log.Infof("Endpoints %s@%s added", pod.Name, pod.Namespace)

				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldEndpoints, ok := old.(*apiv1.Endpoints)
				if !ok {
					log.Errorln(errors.New("Invalid Endpoints object"))
					return
				}
				newEndpoints, ok := new.(*apiv1.Endpoints)
				if !ok {
					log.Errorln(errors.New("Invalid Endpoints object"))
					return
				}

				fmt.Println(oldEndpoints.Name, newEndpoints.Name)

			},
			DeleteFunc: func(obj interface{}) {
				if pod, ok := obj.(*apiv1.Endpoints); ok {
					log.Infof("Endpoints %s@%s deleted", pod.Name, pod.Namespace)

				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
