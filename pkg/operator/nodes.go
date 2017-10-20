package operator

import (
	"context"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/ingress"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initNodeWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Nodes().Watch(metav1.ListOptions{})
		},
	}

	handler := func(obj interface{}) {
		if node, ok := obj.(*apiv1.Node); ok {
			ctx := etx.Background()

			ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
			if err == nil {
				for _, ing := range ingresses.Items {
					engress, err := api.NewEngressFromIngress(ing)
					if err != nil {
						return
					}
					op.updateFirewall(ctx, engress, node)
				}
			}
			engresses, err := op.VoyagerClient.Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
			if err == nil {
				for _, engress := range engresses.Items {
					op.updateFirewall(ctx, &engress, node)
				}
			}
		}
	}

	indexer, informer := cache.NewInformer(lw,
		&apiv1.Node{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handler,
			DeleteFunc: handler,
		},
	)
	// https://github.com/kubernetes/client-go/blob/42a124578af9e61f5c6902fa7b6b2cb6538f17d2/examples/workqueue/main.go#L203
	if nodes, err := op.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{}); err == nil {
		for _, node := range nodes.Items {
			indexer.Add(&apiv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: node.Name,
				},
			})
		}
	}

	return informer
}

func (op *Operator) updateFirewall(ctx context.Context, ing *api.Ingress, node *apiv1.Node) {
	if !ing.ShouldHandleIngress(op.Opt.IngressClass) {
		log.New(ctx).Warningf("Skipping ingress %s@%s, as it is not handled by Voyager.", ing.Name, ing.Namespace)
		return
	}
	t := ing.LBType()
	if t == api.LBTypeLoadBalancer || t == api.LBTypeInternal {
		return
	} else if t == api.LBTypeHostPort {
		if selector := labels.SelectorFromSet(ing.NodeSelector()); !selector.Matches(labels.Set(node.Labels)) {
			return
		}
	}

	ctrl := ingress.NewController(ctx, op.KubeClient, op.CRDClient, op.VoyagerClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, ing)
	if svc, err := op.ServiceLister.Services(ing.Namespace).Get(ing.OffshootName()); err == nil {
		ctrl.EnsureFirewall(svc)
	} else {
		op.recorder.Eventf(
			ing,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to update firewall, Reason: %s",
			err.Error(),
		)
	}
}
