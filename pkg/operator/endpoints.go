package operator

import (
	"reflect"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initEndpointWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Endpoints(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Endpoints(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	indexer, informer := cache.NewIndexerInformer(lw,
		&apiv1.Endpoints{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				ctx := etx.Background()
				logger := log.New(ctx)

				oldEndpoints, ok := old.(*apiv1.Endpoints)
				if !ok {
					logger.Errorln("invalid Endpoints object")
					return
				}
				newEndpoints, ok := new.(*apiv1.Endpoints)
				if !ok {
					logger.Errorln("invalid Endpoints object")
					return
				}

				if reflect.DeepEqual(oldEndpoints.Subsets, newEndpoints.Subsets) {
					return
				}

				// Checking if this endpoint have a service or not. If
				// this do not have a Service we do not want to update our ingress
				svc, err := op.ServiceLister.Services(newEndpoints.Namespace).Get(newEndpoints.Name)
				if err != nil {
					logger.Warningf("Skipping Endpoints %s@%s, as it has no matching service", newEndpoints.Name, newEndpoints.Namespace)
					return
				}
				err = op.updateHAProxyConfig(ctx, svc)
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
