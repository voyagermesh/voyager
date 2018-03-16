package controller

import (
	"strings"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/kutil/tools/queue"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
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

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the secret to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncSecret(key string) error {
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

func (c *Controller) projectTLSSecret(r *core.Secret, projections map[string]ioutilz.FileProjection) error {
	pemKey, found := r.Data[core.TLSPrivateKeyKey]
	if !found {
		return errors.Errorf("secret %s/%s is missing tls.key", c.options.IngressRef.Namespace, r.Name)
	}

	pemCrt, found := r.Data[core.TLSCertKey]
	if !found {
		return errors.Errorf("secret %s/%s is missing tls.crt", c.options.IngressRef.Namespace, r.Name)
	}

	projections["tls/"+r.Name+".pem"] = ioutilz.FileProjection{Mode: 0755, Data: certificateToPEMData(pemCrt, pemKey)}
	return nil
}

func (c *Controller) projectAuthSecret(r *core.Secret, projections map[string]ioutilz.FileProjection) error {
	ca, found := r.Data["ca.crt"]
	if !found {
		return errors.Errorf("secret %s/%s is missing ca.crt", c.options.IngressRef.Namespace, r.Name)
	}
	projections["ca/"+r.Name+"-ca.crt"] = ioutilz.FileProjection{Mode: 0755, Data: ca}

	crl, found := r.Data["crl.pem"]
	if found {
		projections["ca/"+r.Name+"-crl.pem"] = ioutilz.FileProjection{Mode: 0755, Data: crl}
	}
	return nil
}
