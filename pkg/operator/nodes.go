package operator

import (
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initNodeWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Nodes().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Nodes().Watch(options)
		},
	}

	handler := func(obj interface{}) {
		if node, ok := obj.(*core.Node); ok {
			items, err := op.listIngresses()
			if err == nil {
				for _, ing := range items {
					op.updateFirewall(&ing, node)
				}
			}
		}
	}

	op.nodeIndexer, op.nodeInformer = cache.NewIndexerInformer(lw, &core.Node{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: handler,
		// UpdateFunc: func(old interface{}, new interface{}) {},
		DeleteFunc: handler,
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.nodeLister = core_listers.NewNodeLister(op.nodeIndexer)

	// Warm up the cache for initial synchronization
	if nodes, err := op.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{}); err == nil {
		for _, node := range nodes.Items {
			op.nodeIndexer.Add(&core.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: node.Name,
				},
			})
		}
	}
}

func (op *Operator) updateFirewall(ing *api.Ingress, node *core.Node) {
	if !ing.ShouldHandleIngress(op.Opt.IngressClass) {
		return
	}

	switch ing.LBType() {
	case api.LBTypeLoadBalancer, api.LBTypeInternal:
		return
	case api.LBTypeHostPort:
		if selector := labels.SelectorFromSet(ing.NodeSelector()); !selector.Matches(labels.Set(node.Labels)) {
			return
		}
	}

	if key, err := cache.MetaNamespaceKeyFunc(ing); err == nil {
		op.engQueue.Add(key)
		log.Infof("Add/Delete/Update of Node %s, Ingress %s re-queued for update", node.Name, key)
	} else {
		log.Infof("Add/Delete/Update of Node %s, failed to re-queue Ingress %s, reason", node.Name, ing.Name, err.Error())
	}
}
