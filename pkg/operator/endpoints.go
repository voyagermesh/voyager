package operator

import (
	"reflect"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/queue"
)

func (op *Operator) initEndpointWatcher() {
	op.epInformer = op.kubeInformerFactory.Core().V1().Endpoints().Informer()
	op.epQueue = queue.New("Endpoint", op.MaxNumRequeues, op.NumThreads, op.reconcileEndpoint)
	op.epInformer.AddEventHandler(queue.NewEventHandler(op.epQueue.GetQueue(), func(old interface{}, new interface{}) bool {
		oldEndpoints, ok := old.(*core.Endpoints)
		if !ok {
			return false
		}
		newEndpoints, ok := new.(*core.Endpoints)
		if !ok {
			return false
		}
		return !reflect.DeepEqual(oldEndpoints.Subsets, newEndpoints.Subsets)
	}))
	op.epLister = op.kubeInformerFactory.Core().V1().Endpoints().Lister()
}

func (op *Operator) reconcileEndpoint(key string) error {
	obj, exists, err := op.epInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if exists {
		glog.Infof("Sync/Add/Update for Endpoints %s\n", key)
		ep := obj.(*core.Endpoints).DeepCopy()
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
