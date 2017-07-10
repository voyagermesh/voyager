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
func (c *Operator) WatchDaemonSets() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.DaemonSet{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if pod, ok := obj.(*extensions.DaemonSet); ok {
					log.Infof("DaemonSet %s@%s added", pod.Name, pod.Namespace)

				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldDaemonSet, ok := old.(*extensions.DaemonSet)
				if !ok {
					log.Errorln(errors.New("Invalid DaemonSet object"))
					return
				}
				newDaemonSet, ok := new.(*extensions.DaemonSet)
				if !ok {
					log.Errorln(errors.New("Invalid DaemonSet object"))
					return
				}

				fmt.Println(oldDaemonSet.Name, newDaemonSet.Name)

			},
			DeleteFunc: func(obj interface{}) {
				if pod, ok := obj.(*extensions.DaemonSet); ok {
					log.Infof("DaemonSet %s@%s deleted", pod.Name, pod.Namespace)

				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}
