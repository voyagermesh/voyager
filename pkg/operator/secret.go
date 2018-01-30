package operator

import (
	"reflect"

	"github.com/appscode/go/log"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initSecretWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.CoreV1().Secrets(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Secrets(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.secretQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "secret")

	op.secretIndexer, op.secretInformer = cache.NewIndexerInformer(lw, &core.Secret{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				op.secretQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			oldSecret := old.(*core.Secret)
			newSecret := new.(*core.Secret)
			if reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
				return
			}
			if key, err := cache.MetaNamespaceKeyFunc(new); err == nil {
				op.secretQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.secretLister = core_listers.NewSecretLister(op.secretIndexer)
}

func (op *Operator) runSecretWatcher() {
	for op.processNextSecret() {
	}
}

func (op *Operator) processNextSecret() bool {
	key, quit := op.secretQueue.Get()
	if quit {
		return false
	}
	defer op.secretQueue.Done(key)

	err := op.reconcileSecret(key.(string))
	if err == nil {
		op.secretQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Secret %v. Reason: %s", key, err)

	if op.secretQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing Secret %v: %v", key, err)
		op.secretQueue.AddRateLimited(key)
		return true
	}

	op.secretQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping Secret %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) reconcileSecret(key string) error {
	obj, exists, err := op.secretIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if exists {
		glog.Infof("Sync/Add/Update for Secret %s\n", key)
		secret := obj.(*core.Secret).DeepCopy()
		// Secret DataChanged. We need to list all Ingress and check which of
		// those ingress uses this secret as basic auth secret.
		items, err := op.listIngresses()
		if err != nil {
			return err
		}
		for i := range items {
			ing := &items[i]
			if ing.DeletionTimestamp == nil &&
				(ing.ShouldHandleIngress(op.Opt.IngressClass) || op.IngressServiceUsesAuthSecret(ing, secret)) {
				if ing.UsesAuthSecret(secret.Namespace, secret.Name) {
					if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
						return err
					} else {
						op.engQueue.Add(key)
						log.Infof("Add/Delete/Update of secret %s/%s, Ingress %s re-queued for update", secret.Namespace, secret.Name, key)
					}
				}
			}
		}
	}
	return nil
}

func (op *Operator) IngressServiceUsesAuthSecret(ing *tapi.Ingress, secret *core.Secret) bool {
	svcs, err := op.svcLister.List(labels.Everything())
	if err != nil {
		log.Errorln(err)
		return false
	}

	for _, svc := range svcs {
		if ing.HasBackendService(svc.Name, svc.Namespace) {
			if svc.Annotations != nil {
				if svc.Annotations[tapi.AuthSecret] == secret.Name && svc.Namespace == secret.Namespace {
					return true
				}
			}
		}
	}
	return false
}
