/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	promapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/queue"
	prom_util "kmodules.xyz/monitoring-agent-api/prometheus/v1"
)

func (op *Operator) initServiceMonitorWatcher() {
	if !discovery.IsPreferredAPIResource(op.KubeClient.Discovery(), prom_util.SchemeGroupVersion.String(), promapi.ServiceMonitorsKind) {
		klog.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", prom_util.SchemeGroupVersion.String(), promapi.ServiceMonitorsKind)
		return
	}

	op.smonInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return op.PromClient.ServiceMonitors(op.WatchNamespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return op.PromClient.ServiceMonitors(op.WatchNamespace).Watch(context.TODO(), options)
			},
		},
		&promapi.ServiceMonitor{}, op.ResyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	op.smonQueue = queue.New("ServiceMonitor", op.MaxNumRequeues, op.NumThreads, op.reconcileServiceMonitor)
	op.smonInformer.AddEventHandler(queue.NewDeleteHandler(op.smonQueue.GetQueue()))
}

func (op *Operator) reconcileServiceMonitor(key string) error {
	_, exists, err := op.smonInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("ServiceMonitor %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreServiceMonitor(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes service-monitor
func (op *Operator) restoreServiceMonitor(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.StatsServiceName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				klog.Infof("Add/Delete/Update of service-monitor %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}

func (op *Operator) getIngressQueue(apiVersion string) workqueue.RateLimitingInterface {
	if apiVersion == api.APISchemaIngress {
		return op.ingQueue.GetQueue()
	}
	return op.engQueue.GetQueue()
}
