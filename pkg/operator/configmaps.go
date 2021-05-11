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
)

func (op *Operator) initConfigMapWatcher() {
	op.cfgInformer = op.kubeInformerFactory.Core().V1().ConfigMaps().Informer()
	op.cfgQueue = queue.New("ConfigMap", op.MaxNumRequeues, op.NumThreads, op.reconcileConfigMap)
	op.cfgInformer.AddEventHandler(queue.NewDeleteHandler(op.cfgQueue.GetQueue()))
	op.cfgLister = op.kubeInformerFactory.Core().V1().ConfigMaps().Lister()
}

func (op *Operator) reconcileConfigMap(key string) error {
	_, exists, err := op.cfgInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("ConfigMap %s does not exist anymore\n", key)
		if ns, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			return op.restoreConfigMap(name, ns)
		}
	}
	return nil
}

// requeue ingress if user deletes haproxy-configmap
func (op *Operator) restoreConfigMap(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.Namespace == ns &&
			ing.OffshootName() == name {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
				return err
			} else {
				op.getIngressQueue(ing.APISchema()).Add(key)
				klog.Infof("Add/Delete/Update of haproxy configmap %s/%s, Ingress %s re-queued for update", ns, name, key)
				break
			}
		}
	}
	return nil
}
