package operator

import (
	"errors"
	"reflect"

	"github.com/appscode/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initEndpointWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Endpoints(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Endpoints(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	indexer, informer := cache.NewIndexerInformer(lw,
		&apiv1.Endpoints{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
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

				if reflect.DeepEqual(oldEndpoints.Subsets, newEndpoints.Subsets) {
					return
				}

				// Checking if this endpoint have a service or not. If
				// this do not have a Service we do not want to update our ingress
				svc, err := op.ServiceLister.Services(newEndpoints.Namespace).Get(newEndpoints.Name)
				if err != nil {
					log.Warningf("Skipping Endpoints %s@%s, as it has no matching service", newEndpoints.Name, newEndpoints.Namespace)
					return
				}
				err = op.updateHAProxyConfig(svc)
				if err != nil {
					log.Errorln(err)
				}
			},
		},
		cache.Indexers{},
	)
	op.EndpointsLister = core_listers.NewEndpointsLister(indexer)
	return informer
}
