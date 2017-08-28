package operator

import (
	"github.com/appscode/kutil"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initServiceMonitorWatcher() cache.Controller {
	if !kutil.IsPreferredAPIResource(op.KubeClient, prom.TPRGroup+"/"+prom.TPRVersion, prom.TPRServiceMonitorsKind) {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", prom.TPRGroup+"/"+prom.TPRVersion, prom.TPRServiceMonitorsKind)
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
		op.Opt.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if svcmon, ok := obj.(*prom.ServiceMonitor); ok {
					log.Infof("ServiceMonitor %s@%s deleted", svcmon.Name, svcmon.Namespace)
					op.restoreServiceMonitorIfRequired(svcmon)
				}
			},
		},
	)
	return informer
}

func (op *Operator) restoreServiceMonitorIfRequired(svcmon *prom.ServiceMonitor) error {
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
	log.Infof("ServiceMonitor %s@%s requires restoration", svcmon.Name, svcmon.Namespace)
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
