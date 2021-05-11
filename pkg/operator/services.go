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
	_ "voyagermesh.dev/voyager/third_party/forked/cloudprovider/providers"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
)

func (op *Operator) initServiceWatcher() {
	op.svcInformer = op.kubeInformerFactory.Core().V1().Services().Informer()
	op.svcQueue = queue.New("Service", op.MaxNumRequeues, op.NumThreads, op.reconcileService)
	op.svcInformer.AddEventHandler(queue.DefaultEventHandler(op.svcQueue.GetQueue()))
	op.svcLister = op.kubeInformerFactory.Core().V1().Services().Lister()
}

func (op *Operator) reconcileService(key string) error {
	obj, exists, err := op.svcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("Service %s does not exist anymore\n", key)
		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		if isOffshoot, err := op.restoreIngressService(name, ns); isOffshoot {
			return err // assume offshoot service can't be backend service
		} else {
			return op.updateHAProxyConfig(name, ns)
		}
	} else {
		svc := obj.(*core.Service).DeepCopy()
		klog.Infof("Sync/Add/Update for Service %s\n", svc.GetName())
		return op.updateHAProxyConfig(svc.Name, svc.Namespace)
	}
}

// requeue ingress if offshoot-service deleted
// return true if service is offshoot for any ingress
func (op *Operator) restoreIngressService(name, ns string) (bool, error) {
	items, err := op.listIngresses()
	if err == nil {
		for i := range items {
			ing := &items[i]
			if ing.DeletionTimestamp == nil &&
				ing.ShouldHandleIngress(op.IngressClass) &&
				ing.Namespace == ns &&
				ing.OffshootName() == name {
				key, err := cache.MetaNamespaceKeyFunc(ing)
				if err == nil {
					op.getIngressQueue(ing.APISchema()).Add(key)
					klog.Infof("Add/Delete/Update of offshoot service %s/%s, Ingress %s re-queued for update", ns, name, key)
				}
				return true, err
			}
		}
	}
	return false, err
}

// requeue ingress if add/delete/update of backend-service
func (op *Operator) updateHAProxyConfig(name, ns string) error {
	items, err := op.listIngresses()
	if err != nil {
		return err
	}
	for i := range items {
		ing := &items[i]
		if ing.DeletionTimestamp == nil &&
			ing.ShouldHandleIngress(op.IngressClass) &&
			ing.HasBackendService(name, ns) {
			if key, err := cache.MetaNamespaceKeyFunc(ing); err == nil {
				op.getIngressQueue(ing.APISchema()).Add(key)
				klog.Infof("Add/Delete/Update of backend service %s/%s, Ingress %s re-queued for update", ns, name, key)
			}
		}
	}
	return nil
}
