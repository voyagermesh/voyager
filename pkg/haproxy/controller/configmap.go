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
	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	"github.com/pkg/errors"
	atomic_writer "gomodules.xyz/atomic-writer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *Controller) initConfigMapWatcher() {
	c.cfgInformer = c.kubeInformerFactory.Core().V1().ConfigMaps().Informer()
	c.cfgQueue = queue.New("ConfigMap", c.options.MaxNumRequeues, c.options.NumThreads, c.syncConfigMap)
	c.cfgInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*core.ConfigMap); ok {
				if c.isConfigMapUsedInIngress(r) {
					queue.Enqueue(c.cfgQueue.GetQueue(), obj)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if r, ok := newObj.(*core.ConfigMap); ok {
				if c.isConfigMapUsedInIngress(r) {
					queue.Enqueue(c.cfgQueue.GetQueue(), oldObj)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.cfgQueue.GetQueue(), obj)
		},
	})
}

func (c *Controller) isConfigMapUsedInIngress(s *core.ConfigMap) bool {
	return s.Name == api.VoyagerPrefix+c.options.IngressRef.Name // Ingress.OffshootName()
}

func (c *Controller) syncConfigMap(_ string) error {
	key, err := cache.MetaNamespaceKeyFunc(cache.ExplicitKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name))
	if err != nil {
		return err
	}
	c.getIngressWorker().GetQueue().Add(key)
	return nil
}

func (c *Controller) getConfigMap(name string) (*core.ConfigMap, error) {
	obj, exists, err := c.cfgInformer.GetIndexer().GetByKey(c.options.IngressRef.Namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(core.Resource("secret"), name)
	}
	return obj.(*core.ConfigMap), nil
}

func (c *Controller) projectHAProxyConfig(r *core.ConfigMap, projections map[string]atomic_writer.FileProjection) error {
	cfg, found := r.Data["haproxy.cfg"]
	if !found {
		return errors.Errorf("configmap %s/%s is missing haproxy.cfg", c.options.IngressRef.Namespace, r.Name)
	}
	projections["haproxy.cfg"] = atomic_writer.FileProjection{Mode: 0755, Data: []byte(cfg)}
	return nil
}
