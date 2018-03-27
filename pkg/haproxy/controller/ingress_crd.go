package controller

import (
	"fmt"
	"strings"
	"time"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	voyager_informers "github.com/appscode/voyager/client/informers/externalversions/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
)

func (c *Controller) initIngressCRDWatcher() {
	// https://github.com/kubernetes/kubernetes/issues/51046
	c.engInformer = c.voyagerInformerFactory.InformerFor(&api.Ingress{}, func(client cs.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return voyager_informers.NewIngressInformer(
			client,
			c.options.IngressRef.Namespace,
			resyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)
	})
	c.engQueue = queue.New("IngressCRD", c.options.MaxNumRequeues, c.options.NumThreads, c.syncIngressCRD)
	c.engInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				r.Migrate()
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					queue.Enqueue(c.engQueue.GetQueue(), obj)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if r, ok := newObj.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				r.Migrate()
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					queue.Enqueue(c.engQueue.GetQueue(), newObj)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				queue.Enqueue(c.engQueue.GetQueue(), obj)
			}
		},
	})
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncIngressCRD(key string) error {
	obj, exists, err := c.engInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Ingress, so that we will see a delete for one d
		fmt.Printf("Ingress %s does not exist anymore\n", key)
	} else {
		d := obj.(*api.Ingress)
		fmt.Printf("Sync/Add/Update for Ingress %s\n", d.GetName())
		d.Migrate()

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

func (c *Controller) getIngress() (*api.Ingress, error) {
	if c.options.UsesEngress() {
		obj, exists, err := c.engInformer.GetIndexer().GetByKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, kerr.NewNotFound(api.Resource("ingress"), c.options.IngressRef.Name)
		}
		i := obj.(*api.Ingress)
		i.Migrate()
		err = i.IsValid(c.options.CloudProvider)
		if err != nil {
			return nil, err
		}
		return i, nil
	}

	obj, exists, err := c.ingInformer.GetIndexer().GetByKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerr.NewNotFound(extensions.Resource("ingress"), c.options.IngressRef.Name)
	}
	i, err := api.NewEngressFromIngress(obj)
	if err != nil {
		return nil, err
	}
	err = i.IsValid(c.options.CloudProvider)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (c *Controller) projectConfig(ing *api.Ingress, projections map[string]ioutilz.FileProjection) error {
	r, err := c.getConfigMap(api.VoyagerPrefix + ing.Name)
	if err != nil {
		return err
	}
	return c.projectHAProxyConfig(r, projections)
}

func (c *Controller) projectCerts(ing *api.Ingress, projections map[string]ioutilz.FileProjection) error {
	r, err := c.getConfigMap(api.VoyagerPrefix + ing.Name)
	if err != nil {
		return err
	}
	err = c.projectHAProxyConfig(r, projections)
	if err != nil {
		return err
	}

	for _, tls := range ing.Spec.TLS {
		if strings.EqualFold(tls.Ref.Kind, api.ResourceKindCertificate) {
			r, err := c.getCertificate(tls.Ref.Name)
			if err != nil {
				return err
			}
			err = c.projectCertificate(r, projections)
			if err != nil {
				return err
			}
		} else {
			r, err := c.getSecret(tls.Ref.Name)
			if err != nil {
				return err
			}
			err = c.projectTLSSecret(r, projections)
			if err != nil {
				return err
			}
		}
	}

	if name := ing.AuthTLSSecret(); name != "" {
		r, err := c.getSecret(name)
		if err != nil {
			return err
		}
		err = c.projectAuthSecret(r, projections)
		if err != nil {
			return err
		}
	} else {
		for _, fr := range ing.Spec.FrontendRules {
			if fr.Auth != nil && fr.Auth.TLS != nil {
				r, err := c.getSecret(fr.Auth.TLS.SecretName)
				if err != nil {
					return err
				}
				err = c.projectAuthSecret(r, projections)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Controller) mountIngress(ing *api.Ingress) error {
	cfgProjections := map[string]ioutilz.FileProjection{}
	err := c.projectConfig(ing, cfgProjections)
	if err != nil {
		return err
	}
	cfgChanged, err := c.cfgWriter.Write(cfgProjections)
	if err != nil {
		return err
	}
	if cfgChanged {
		incConfigChangedCounter()
	}

	certProjections := map[string]ioutilz.FileProjection{}
	err = c.projectCerts(ing, certProjections)
	if err != nil {
		return err
	}
	certChanged, err := c.certWriter.Write(certProjections)
	if err != nil {
		return err
	}
	if certChanged {
		incCertChangedCounter()
	}

	if cfgChanged || certChanged {
		return runCmd()
	}
	return nil
}
