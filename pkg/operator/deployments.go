package operator

import (
	"github.com/appscode/go/log"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initDeploymentWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.KubeClient.AppsV1beta1().Deployments(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.AppsV1beta1().Deployments(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.dpQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment")

	op.dpIndexer, op.dpInformer = cache.NewIndexerInformer(lw, &apps.Deployment{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				op.dpQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.dpLister = apps_listers.NewDeploymentLister(op.dpIndexer)
}

func (op *Operator) runDeploymentWatcher() {
	for op.processNextDeployment() {
	}
}

func (op *Operator) processNextDeployment() bool {
	key, quit := op.dpQueue.Get()
	if quit {
		return false
	}
	defer op.dpQueue.Done(key)

	err := op.runDeploymentInjector(key.(string))
	if err == nil {
		op.dpQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Deployment %v. Reason: %s", key, err)

	if op.dpQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)
		op.dpQueue.AddRateLimited(key)
		return true
	}

	op.dpQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) runDeploymentInjector(key string) error {
	_, exists, err := op.dpIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Deployment %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreDeploymentIfRequired(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-deployment
func (op *Operator) restoreDeploymentIfRequired(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) && engress.Namespace == ns && engress.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(engress); err != nil {
				return err
			} else {
				op.engQueue.Add(key)
				log.Infof("Add/Delete/Update of haproxy deployment %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
