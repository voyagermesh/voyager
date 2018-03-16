package controller

import (
	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
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

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the configmap to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncConfigMap(key string) error {
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

func (c *Controller) projectHAProxyConfig(r *core.ConfigMap, projections map[string]ioutilz.FileProjection) error {
	cfg, found := r.Data["haproxy.cfg"]
	if !found {
		return errors.Errorf("configmap %s/%s is missing haproxy.cfg", c.options.IngressRef.Namespace, r.Name)
	}
	projections["haproxy.cfg"] = ioutilz.FileProjection{Mode: 0755, Data: []byte(cfg)}
	return nil
}
