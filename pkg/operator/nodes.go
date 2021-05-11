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

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (op *Operator) initNodeWatcher() {
	handler := func(obj interface{}) {
		if node, ok := obj.(*core.Node); ok {
			items, err := op.listIngresses()
			if err == nil {
				for _, ing := range items {
					op.updateFirewall(&ing, node)
				}
			}
		}
	}

	op.nodeInformer = op.kubeInformerFactory.Core().V1().Nodes().Informer()
	op.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: handler,
		// UpdateFunc: func(old interface{}, new interface{}) {},
		DeleteFunc: handler,
	})
	op.nodeLister = op.kubeInformerFactory.Core().V1().Nodes().Lister()

	// Warm up the cache for initial synchronization
	if nodes, err := op.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, node := range nodes.Items {
			err = op.nodeInformer.GetIndexer().Add(&core.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: node.Name,
				},
			})
			klog.Error(err)
		}
	}
}

func (op *Operator) updateFirewall(ing *api.Ingress, node *core.Node) {
	if !ing.ShouldHandleIngress(op.IngressClass) {
		return
	}

	switch ing.LBType() {
	case api.LBTypeLoadBalancer, api.LBTypeInternal:
		return
	case api.LBTypeHostPort:
		if selector := labels.SelectorFromSet(ing.Spec.NodeSelector); !selector.Matches(labels.Set(node.Labels)) {
			return
		}
	}

	if key, err := cache.MetaNamespaceKeyFunc(ing); err == nil {
		op.getIngressQueue(ing.APISchema()).Add(key)
		klog.Infof("Add/Delete/Update of Node %s, Ingress %s re-queued for update", node.Name, key)
	} else {
		klog.Infof("Add/Delete/Update of Node %s, failed to re-queue Ingress %s, reason %s", node.Name, ing.Name, err)
	}
}
