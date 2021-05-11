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
	"reflect"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	_ "voyagermesh.dev/voyager/third_party/forked/cloudprovider/providers"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
)

func (op *Operator) initSecretWatcher() {
	op.secretInformer = op.kubeInformerFactory.Core().V1().Secrets().Informer()
	op.secretQueue = queue.New("Secret", op.MaxNumRequeues, op.NumThreads, op.reconcileSecret)
	op.secretInformer.AddEventHandler(queue.NewEventHandler(op.secretQueue.GetQueue(), func(old interface{}, new interface{}) bool {
		oldSecret := old.(*core.Secret)
		newSecret := new.(*core.Secret)
		return !reflect.DeepEqual(oldSecret.Data, newSecret.Data)
	}))
	op.secretLister = op.kubeInformerFactory.Core().V1().Secrets().Lister()
}

func (op *Operator) reconcileSecret(key string) error {
	obj, exists, err := op.secretInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if exists {
		klog.Infof("Sync/Add/Update for Secret %s\n", key)
		secret := obj.(*core.Secret).DeepCopy()
		// Secret DataChanged. We need to list all Ingress and check which of
		// those ingress uses this secret as basic auth secret.
		items, err := op.listIngresses()
		if err != nil {
			return err
		}
		for i := range items {
			ing := &items[i]
			if ing.DeletionTimestamp == nil &&
				(ing.ShouldHandleIngress(op.IngressClass) || op.IngressServiceUsesAuthSecret(ing, secret)) {
				if ing.UsesAuthSecret(secret.Namespace, secret.Name) {
					if key, err := cache.MetaNamespaceKeyFunc(ing); err != nil {
						return err
					} else {
						op.getIngressQueue(ing.APISchema()).Add(key)
						klog.Infof("Add/Delete/Update of secret %s/%s, Ingress %s re-queued for update", secret.Namespace, secret.Name, key)
					}
				}
			}
		}
	}
	return nil
}

func (op *Operator) IngressServiceUsesAuthSecret(ing *api.Ingress, secret *core.Secret) bool {
	svcs, err := op.svcLister.List(labels.Everything())
	if err != nil {
		klog.Errorln(err)
		return false
	}

	for _, svc := range svcs {
		if ing.HasBackendService(svc.Name, svc.Namespace) {
			if svc.Annotations != nil {
				if svc.Annotations[api.AuthSecret] == secret.Name && svc.Namespace == secret.Namespace {
					return true
				}
			}
		}
	}
	return false
}
