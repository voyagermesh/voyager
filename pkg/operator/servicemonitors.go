package operator

import (
	"context"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initServiceMonitorWatcher() cache.Controller {
	if !kutil.IsPreferredAPIResource(op.KubeClient, prom.Group+"/"+prom.Version, prom.ServiceMonitorsKind) {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", prom.Group+"/"+prom.Version, prom.ServiceMonitorsKind)
		return nil
	}

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.PromClient.ServiceMonitors(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.PromClient.ServiceMonitors(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&prom.ServiceMonitor{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				ctx := etx.Background()
				if svcmon, ok := obj.(*prom.ServiceMonitor); ok {
					log.New(ctx).Infof("ServiceMonitor %s@%s deleted", svcmon.Name, svcmon.Namespace)
					op.restoreServiceMonitorIfRequired(ctx, svcmon)
				}
			},
		},
	)
	return informer
}

func (op *Operator) restoreServiceMonitorIfRequired(ctx context.Context, svcmon *prom.ServiceMonitor) error {
	if svcmon.Annotations == nil {
		return nil
	}

	// deleted resource have source reference
	engress, err := op.findOrigin(svcmon.ObjectMeta)
	if err != nil {
		return err
	}
	monSpec, err := engress.MonitorSpec()
	if err != nil {
		return err
	}
	if monSpec == nil && monSpec.Prometheus == nil {
		return nil
	}

	// Ingress Still exists, restore resource
	log.New(ctx).Infof("ServiceMonitor %s@%s requires restoration", svcmon.Name, svcmon.Namespace)
	svcmon.SelfLink = ""
	svcmon.ResourceVersion = ""
	// Old resource and annotations are missing so we need to add the annotations
	if svcmon.Annotations == nil {
		svcmon.Annotations = make(map[string]string)
	}
	svcmon.Annotations[api.OriginAPISchema] = engress.APISchema()
	svcmon.Annotations[api.OriginName] = engress.Name

	_, err = op.PromClient.ServiceMonitors(svcmon.Namespace).Create(svcmon)
	return err
}
