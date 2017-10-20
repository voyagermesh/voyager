package operator

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
// ref: https://github.com/kubernetes/kubernetes/issues/46736
func (op *Operator) initNamespaceWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Namespaces().Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&apiv1.Namespace{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if ns, ok := obj.(*apiv1.Namespace); ok {
					if resources, err := op.VoyagerClient.Certificates(ns.Name).List(metav1.ListOptions{}); err == nil {
						for _, resource := range resources.Items {
							op.VoyagerClient.Certificates(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
						}
					}
					if resources, err := op.VoyagerClient.Ingresses(ns.Name).List(metav1.ListOptions{}); err == nil {
						for _, resource := range resources.Items {
							op.VoyagerClient.Ingresses(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
						}
					}
				}
			},
		},
	)
	return informer
}
