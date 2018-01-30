package operator

import (
	"fmt"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
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

func (op *Operator) initServiceWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.CoreV1().Services(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Services(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.svcQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "service")

	op.svcIndexer, op.svcInformer = cache.NewIndexerInformer(lw, &core.Service{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				op.svcQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				op.svcQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				op.svcQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.svcLister = core_listers.NewServiceLister(op.svcIndexer)
}

func (op *Operator) runServiceWatcher() {
	for op.processNextService() {
	}
}

func (op *Operator) processNextService() bool {
	key, quit := op.svcQueue.Get()
	if quit {
		return false
	}
	defer op.svcQueue.Done(key)

	err := op.reconcileService(key.(string))
	if err == nil {
		op.svcQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Service %v. Reason: %s", key, err)

	if op.svcQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing service %v: %v", key, err)
		op.svcQueue.AddRateLimited(key)
		return true
	}

	op.svcQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping service %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) reconcileService(key string) error {
	obj, exists, err := op.svcIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Service %s does not exist anymore\n", key)
		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		if isOffshoot, err := op.restoreServiceIfRequired(name, ns); isOffshoot {
			return err // assume offshoot service can't be backend service
		} else {
			return op.updateHAProxyConfig(name, ns)
		}
	} else {
		svc := obj.(*core.Service)
		glog.Infof("Sync/Add/Update for Service %s\n", svc.GetName())
		return op.updateHAProxyConfig(svc.Name, svc.Namespace)
	}
	return nil
}

// requeue ingress if offshoot-service deleted
// return true if service is offshoot for any ingress
func (op *Operator) restoreServiceIfRequired(name, ns string) (bool, error) {
	items, err := op.listIngresses()
	if err == nil {
		for i := range items {
			engress := &items[i]
			if engress.ShouldHandleIngress(op.Opt.IngressClass) && engress.Namespace == ns && engress.OffshootName() == name {
				key, err := cache.MetaNamespaceKeyFunc(engress)
				if err == nil {
					op.engQueue.Add(key)
					log.Infof("Add/Delete/Update of offshoot service %s/%s, Ingress %s re-queued for update", ns, name, key)
				}
				return true, err
			}
		}
	}
	return false, err
}

// requeue ingress if add/delete/update of backend-service
func (op *Operator) updateHAProxyConfig(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) && engress.HasBackendService(name, ns) {
			if key, err := cache.MetaNamespaceKeyFunc(engress); err == nil {
				op.engQueue.Add(key)
				log.Infof("Add/Delete/Update of backend service %s/%s, Ingress %s re-queued for update", ns, name, key)
			}
		}
	}
	return nil
}

func (op *Operator) findOrigin(meta metav1.ObjectMeta) (*tapi.Ingress, error) {
	if meta.Annotations == nil {
		return nil, nil
	}

	sourceName, sourceNameFound := meta.Annotations[tapi.OriginName]
	sourceType, sourceTypeFound := meta.Annotations[tapi.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		return nil, errors.New("no Types or Name found").Err()
	}

	if sourceType == tapi.APISchemaIngress {
		ingress, err := op.KubeClient.ExtensionsV1beta1().Ingresses(meta.Namespace).Get(sourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return tapi.NewEngressFromIngress(ingress)
	} else if sourceType == tapi.APISchemaEngress {
		return op.VoyagerClient.Ingresses(meta.Namespace).Get(sourceName, metav1.GetOptions{})
	}
	return nil, fmt.Errorf("unknown ingress type %s", sourceType)
}
