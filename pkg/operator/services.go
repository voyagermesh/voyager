package operator

import (
	"context"
	"fmt"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/ingress"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	apiv1 "k8s.io/api/core/v1"
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
		&apiv1.Service{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ctx := etx.Background()
				if svc, ok := obj.(*apiv1.Service); ok {
					log.New(ctx).Infof("Service %s@%s added", svc.Name, svc.Namespace)
					op.updateHAProxyConfig(ctx, svc)
				}
			},
			DeleteFunc: func(obj interface{}) {
				ctx := etx.Background()
				if svc, ok := obj.(*apiv1.Service); ok {
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

func (op *Operator) restoreServiceIfRequired(ctx context.Context, svc *apiv1.Service) (bool, error) {
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

func (op *Operator) updateHAProxyConfig(ctx context.Context, svc *apiv1.Service) error {
	logger := log.New(ctx)

	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) {
			if engress.HasBackendService(svc.Name, svc.Namespace) {
				ctrl := ingress.NewController(ctx, op.KubeClient, op.CRDClient, op.VoyagerClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, engress)
				if ctrl.IsExists() {
					// Loadbalancer resource for this ingress is found in its place,
					// so no need to create the resources. First trying to update
					// the configMap only for the rules.
					// In case of any failure in soft update we will make hard update
					// to the resource. If hard update encounters errors then we will
					// recreate the resource from scratch.
					logger.Infof("Offshoots of %s Ingress %s/%s exist, trying to update", engress.APISchema(), engress.Namespace, engress.Name)
					cfgErr := ctrl.Update(0, nil)
					if cfgErr != nil {
						logger.Infof("Failed to update offshoots of %s Ingress %s/%s. Reason: %s", engress.APISchema(), engress.Namespace, engress.Name, cfgErr)
					}
				} else {
					logger.Infof("One or more offshoots of %s Ingress %s/%s is missing, trying to create", engress.APISchema(), engress.Namespace, engress.Name)
					ctrl.Create()
				}
			}
		}
	}
	return nil
}
