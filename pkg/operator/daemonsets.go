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
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
	wpi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (op *Operator) initDaemonSetWatcher() {
	op.dmInformer = op.kubeInformerFactory.Apps().V1().DaemonSets().Informer()
	op.dmQueue = queue.New("DaemonSet", op.MaxNumRequeues, op.NumThreads, op.reconcileDaemonSet)
	op.dmInformer.AddEventHandler(queue.NewDeleteHandler(op.dmQueue.GetQueue()))
	op.dmLister = op.kubeInformerFactory.Apps().V1().DaemonSets().Lister()
}

func (op *Operator) reconcileDaemonSet(key string) error {
	_, exists, err := op.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("DaemonSet %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreDaemonSet(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-deployment
func (op *Operator) restoreDaemonSet(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.WorkloadKind() == wpi.KindDaemonSet &&
			ing.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				klog.Infof("Add/Delete/Update of haproxy deployment %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
