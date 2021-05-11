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

	"github.com/pkg/errors"
	atomic_writer "gomodules.xyz/atomic-writer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *Controller) initSecretWatcher() {
	c.secretInformer = c.kubeInformerFactory.Core().V1().Secrets().Informer()
	c.secretQueue = queue.New("Secret", c.options.MaxNumRequeues, c.options.NumThreads, c.syncSecret)
	c.secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*core.Secret); ok {
				if c.isSecretUsedInIngress(r) {
					queue.Enqueue(c.secretQueue.GetQueue(), obj)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if r, ok := newObj.(*core.Secret); ok {
				if c.isSecretUsedInIngress(r) {
					queue.Enqueue(c.secretQueue.GetQueue(), newObj)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.secretQueue.GetQueue(), obj)
		},
	})
}

func (c *Controller) isSecretUsedInIngress(s *core.Secret) bool {
	return c.isSecretUsedForTLSTermination(s) || c.isSecretUsedForTLSAuth(s)
}

func (c *Controller) isSecretUsedForTLSTermination(s *core.Secret) bool {
	if s.Namespace != c.options.IngressRef.Namespace {
		return false
	}
	r, err := c.getIngress()
	if err != nil {
		return false
	}
	for _, tls := range r.Spec.TLS {
		if s.Name == tls.Ref.Name && (strings.EqualFold(tls.Ref.Kind, "Secret") || tls.Ref.Kind == "") {
			return true
		}
	}

	return false
}

func (c *Controller) isSecretUsedForTLSAuth(s *core.Secret) bool {
	if s.Namespace != c.options.IngressRef.Namespace {
		return false
	}
	r, err := c.getIngress()
	if err != nil {
		return false
	}
	for _, fr := range r.Spec.FrontendRules {
		if fr.Auth != nil {
			if fr.Auth.TLS != nil && fr.Auth.TLS.SecretName == s.Name {
				return true
			}
		}
	}

	return false
}

func (c *Controller) syncSecret(_ string) error {
	key, err := cache.MetaNamespaceKeyFunc(cache.ExplicitKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name))
	if err != nil {
		return err
	}
	c.getIngressWorker().GetQueue().Add(key)
	return nil
}

func (c *Controller) getSecret(name string) (*core.Secret, error) {
	obj, exists, err := c.secretInformer.GetIndexer().GetByKey(c.options.IngressRef.Namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(core.Resource("secret"), name)
	}
	return obj.(*core.Secret), nil
}

func (c *Controller) projectTLSSecret(r *core.Secret, projections map[string]atomic_writer.FileProjection) error {
	pemKey, found := r.Data[core.TLSPrivateKeyKey]
	if !found {
		return errors.Errorf("secret %s/%s is missing tls.key", c.options.IngressRef.Namespace, r.Name)
	}

	pemCrt, found := r.Data[core.TLSCertKey]
	if !found {
		return errors.Errorf("secret %s/%s is missing tls.crt", c.options.IngressRef.Namespace, r.Name)
	}

	projections["tls/"+r.Name+".pem"] = atomic_writer.FileProjection{Mode: 0755, Data: certificateToPEMData(pemCrt, pemKey)}
	return nil
}

func (c *Controller) projectAuthSecret(r *core.Secret, projections map[string]atomic_writer.FileProjection) error {
	ca, found := r.Data["ca.crt"]
	if !found {
		return errors.Errorf("secret %s/%s is missing ca.crt", c.options.IngressRef.Namespace, r.Name)
	}
	projections["ca/"+r.Name+"-ca.crt"] = atomic_writer.FileProjection{Mode: 0755, Data: ca}

	crl, found := r.Data["crl.pem"]
	if found {
		projections["ca/"+r.Name+"-crl.pem"] = atomic_writer.FileProjection{Mode: 0755, Data: crl}
	}
	return nil
}
