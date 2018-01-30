package operator

import (
	"reflect"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initEndpointWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.CoreV1().Endpoints(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Endpoints(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.epQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "endpoints")

	op.epIndexer, op.epInformer = cache.NewIndexerInformer(lw, &core.Endpoints{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				op.epQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			oldEndpoints, ok := old.(*core.Endpoints)
			if !ok {
				log.Errorln("invalid Endpoints object")
				return
			}
			newEndpoints, ok := new.(*core.Endpoints)
			if !ok {
				log.Errorln("invalid Endpoints object")
				return
			}
			if reflect.DeepEqual(oldEndpoints.Subsets, newEndpoints.Subsets) {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				op.epQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.epLister = core_listers.NewEndpointsLister(op.epIndexer)
}

func (op *Operator) runEndpointWatcher() {
	for op.processNextEndpoint() {
	}
}

func (op *Operator) processNextEndpoint() bool {
	key, quit := op.epQueue.Get()
	if quit {
		return false
	}
	defer op.epQueue.Done(key)

	err := op.reconcileEndpoint(key.(string))
	if err == nil {
		op.epQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Endpoints %v. Reason: %s", key, err)

	if op.epQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing Endpoints %v: %v", key, err)
		op.epQueue.AddRateLimited(key)
		return true
	}

	op.epQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping Endpoints %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) reconcileEndpoint(key string) error {
	obj, exists, err := op.epIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if exists {
		glog.Infof("Sync/Add/Update for Endpoints %s\n", key)
		ep := obj.(*core.Endpoints)
		// Checking if this endpoint have a service or not. If
		// this do not have a Service we do not want to update our ingress
		svc, err := op.svcLister.Services(ep.Namespace).Get(ep.Name)
		if err != nil {
			log.Warningf("Skipping Endpoints %s/%s, as it has no matching service", ep.Namespace, ep.Name)
			return nil
		}
		return op.updateHAProxyConfig(svc.Name, svc.Namespace)
	}
	return nil
}
