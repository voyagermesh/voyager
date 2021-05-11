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

package controller

import (
	"fmt"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/pkg/eventer"

	core "k8s.io/api/core/v1"
	extension "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ext_informers "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *Controller) initIngressWatcher() {
	c.ingInformer = c.kubeInformerFactory.InformerFor(&extension.Ingress{}, func(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return ext_informers.NewFilteredIngressInformer(
			client,
			c.options.IngressRef.Namespace,
			resyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			func(options *metav1.ListOptions) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String()
			},
		)
	})
	c.ingQueue = queue.New("Ingress", c.options.MaxNumRequeues, c.options.NumThreads, c.syncIngress)
	c.ingInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*extension.Ingress); !ok || r.Name != c.options.IngressRef.Name {
				return
			}
			if r, err := api.NewEngressFromIngress(obj); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					queue.Enqueue(c.ingQueue.GetQueue(), obj)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if r, ok := newObj.(*extension.Ingress); !ok || r.Name != c.options.IngressRef.Name {
				return
			}
			if r, err := api.NewEngressFromIngress(newObj); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					queue.Enqueue(c.ingQueue.GetQueue(), newObj)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if r, ok := obj.(*extension.Ingress); !ok || r.Name != c.options.IngressRef.Name {
				return
			}
			if r, err := api.NewEngressFromIngress(obj); err == nil {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				queue.Enqueue(c.ingQueue.GetQueue(), obj)
			}
		},
	})
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncIngress(key string) error {
	obj, exists, err := c.ingInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Ingress, so that we will see a delete for one d
		fmt.Printf("Ingress %s does not exist anymore\n", key)
	} else {
		d, err := api.NewEngressFromIngress(obj)
		if err != nil {
			return err
		}
		fmt.Printf("Sync/Add/Update for Ingress %s\n", d.GetName())

		err = c.mountIngress(d)
		if err != nil {
			c.recorder.Event(
				d.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressTLSMountFailed,
				err.Error(),
			)
			return err
		}
	}
	return nil
}
