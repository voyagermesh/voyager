package operator

import (
	"context"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initConfigMapWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().ConfigMaps(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().ConfigMaps(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&apiv1.ConfigMap{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if cfgmap, ok := obj.(*apiv1.ConfigMap); ok {
					ctx := etx.Background()
					log.New(ctx).Infof("ConfigMap %s@%s deleted", cfgmap.Name, cfgmap.Namespace)
					op.restoreConfigMapIfRequired(ctx, cfgmap)
				}
			},
		},
	)
	return informer
}

func (op *Operator) restoreConfigMapIfRequired(ctx context.Context, cfgmap *apiv1.ConfigMap) error {
	if cfgmap.Annotations == nil {
		return nil
	}

	// deleted resource have source reference
	engress, err := op.findOrigin(cfgmap.ObjectMeta)
	if err != nil {
		return err
	}

	// Ingress Still exists, restore resource
	log.New(ctx).Infof("ConfigMap %s@%s requires restoration", cfgmap.Name, cfgmap.Namespace)
	cfgmap.SelfLink = ""
	cfgmap.ResourceVersion = ""
	// Old resource and annotations are missing so we need to add the annotations
	if cfgmap.Annotations == nil {
		cfgmap.Annotations = make(map[string]string)
	}
	cfgmap.Annotations[api.OriginAPISchema] = engress.APISchema()
	cfgmap.Annotations[api.OriginName] = engress.Name

	_, err = op.KubeClient.CoreV1().ConfigMaps(cfgmap.Namespace).Create(cfgmap)
	return err
}
