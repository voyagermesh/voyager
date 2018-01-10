package operator

import (
	"context"
	"fmt"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initServiceWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Services(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Services(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	indexer, informer := cache.NewIndexerInformer(lw,
		&core.Service{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ctx := etx.Background()
				if svc, ok := obj.(*core.Service); ok {
					log.New(ctx).Infof("Service %s@%s added", svc.Name, svc.Namespace)
					op.updateHAProxyConfig(ctx, svc)
				}
			},
			DeleteFunc: func(obj interface{}) {
				ctx := etx.Background()
				if svc, ok := obj.(*core.Service); ok {
					log.New(ctx).Infof("Service %s@%s deleted", svc.Name, svc.Namespace)
					if restored, err := op.restoreServiceIfRequired(ctx, svc); err == nil && restored {
						return
					}
					op.updateHAProxyConfig(ctx, svc)
				}
			},
		},
		cache.Indexers{},
	)
	op.ServiceLister = core_listers.NewServiceLister(indexer)
	return informer
}

func (op *Operator) restoreServiceIfRequired(ctx context.Context, svc *core.Service) (bool, error) {
	if svc.Annotations == nil {
		return false, nil
	}

	// deleted resource have source reference
	engress, err := op.findOrigin(svc.ObjectMeta)
	if err != nil {
		return false, err
	}

	if svc.Name == engress.StatsServiceName() && !engress.Stats() {
		return false, nil
	}

	// Ingress Still exists, restore resource
	log.New(ctx).Infof("Service %s@%s requires restoration", svc.Name, svc.Namespace)
	svc.Spec.ClusterIP = "" // Remove cluster IP
	svc.SelfLink = ""
	svc.ResourceVersion = ""
	// Old resource and annotations are missing so we need to add the annotations
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	svc.Annotations[tapi.OriginAPISchema] = engress.APISchema()
	svc.Annotations[tapi.OriginName] = engress.Name

	_, err = op.KubeClient.CoreV1().Services(svc.Namespace).Create(svc)
	return true, err
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

func (op *Operator) updateHAProxyConfig(ctx context.Context, svc *core.Service) error {
	logger := log.New(ctx)

	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) {
			if engress.HasBackendService(svc.Name, svc.Namespace) {
				if key, err := cache.MetaNamespaceKeyFunc(engress); err == nil {
					op.engQueue.Add(key)
					logger.Infof("Add/Delete/Update of backend service %s@%s, Ingress %s re-queued for update", svc.Name, svc.Namespace, key)
				}
			}
		}
	}
	return nil
}
