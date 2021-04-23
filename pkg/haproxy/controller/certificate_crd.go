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
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	ioutilz "gomodules.xyz/x/ioutil"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *Controller) initCertificateCRDWatcher() {
	c.crtInformer = c.voyagerInformerFactory.Voyager().V1beta1().Certificates().Informer()
	c.crtQueue = queue.New("Certificate", c.options.MaxNumRequeues, c.options.NumThreads, c.syncCertificate)
	c.crtInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(r) {
					queue.Enqueue(c.crtQueue.GetQueue(), obj)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if r, ok := newObj.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(r) {
					queue.Enqueue(c.crtQueue.GetQueue(), newObj)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.crtQueue.GetQueue(), obj)
		},
	})
}

func (c *Controller) isCertificateUsedInIngress(s *api.Certificate) bool {
	if s.Namespace != c.options.IngressRef.Namespace {
		return false
	}
	r, err := c.getIngress()
	if err != nil {
		return false
	}
	for _, tls := range r.Spec.TLS {
		if s.Name == tls.Ref.Name && strings.EqualFold(tls.Ref.Kind, api.ResourceKindCertificate) {
			return true
		}
	}
	return false
}

func (c *Controller) syncCertificate(_ string) error {
	key, err := cache.MetaNamespaceKeyFunc(cache.ExplicitKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name))
	if err != nil {
		return err
	}
	c.getIngressWorker().GetQueue().Add(key)
	return nil
}

func (c *Controller) getCertificate(name string) (*api.Certificate, error) {
	obj, exists, err := c.crtInformer.GetIndexer().GetByKey(c.options.IngressRef.Namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(core.Resource("secret"), name)
	}
	return obj.(*api.Certificate), nil
}

func (c *Controller) projectCertificate(r *api.Certificate, projections map[string]ioutilz.FileProjection) error {
	pemCrt, pemKey, err := c.store.Get(r)
	if err != nil {
		return err
	}
	if pemCrt != nil && pemKey != nil {
		projections["tls/"+r.SecretName()+".pem"] = ioutilz.FileProjection{Mode: 0755, Data: certificateToPEMData(pemCrt, pemKey)}
	}
	return nil
}
