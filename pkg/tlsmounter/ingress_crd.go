package tlsmounter

import (
	"fmt"
	"strings"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Controller) initIngressCRDWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.VoyagerClient.Ingresses(c.options.IngressRef.Namespace).List(metav1.ListOptions{
			// https://github.com/kubernetes/kubernetes/issues/51046
			//FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.VoyagerClient.Ingresses(c.options.IngressRef.Namespace).Watch(metav1.ListOptions{
			// https://github.com/kubernetes/kubernetes/issues/51046
			//FieldSelector: fields.OneTermEqualSelector("metadata.name", c.options.IngressRef.Name).String(),
			})
		},
	}

	// create the workqueue
	c.engQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress-crd")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	c.engIndexer, c.engInformer = cache.NewIndexerInformer(lw, &api.Ingress{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				r.Migrate()
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						c.engQueue.Add(key)
					}
				}
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			if r, ok := new.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				r.Migrate()
				if err := r.IsValid(c.options.CloudProvider); err == nil {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						c.engQueue.Add(key)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Ingress); ok {
				if r.Name != c.options.IngressRef.Name {
					return
				}
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.engQueue.Add(key)
				}
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) runIngressCRDWatcher() {
	for c.processNextIngressCRD() {
	}
}

func (c *Controller) processNextIngressCRD() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.engQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.engQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.syncIngressCRD(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.engQueue.Forget(key)
		return true
	}
	log.Errorln("Failed to process Ingress %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.engQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.engQueue.AddRateLimited(key)
		return true
	}

	c.engQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncIngressCRD(key string) error {
	obj, exists, err := c.engIndexer.GetByKey(key)
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

		err = c.mountIngress(d, true)
		if err != nil {
			c.recorder.Event(
				d.ObjectReference(),
				apiv1.EventTypeWarning,
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
		obj, exists, err := c.engIndexer.GetByKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name)
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

	obj, exists, err := c.ingIndexer.GetByKey(c.options.IngressRef.Namespace + "/" + c.options.IngressRef.Name)
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

func (c *Controller) projectIngress(ing *api.Ingress, projections map[string]ioutilz.FileProjection) error {
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

func (c *Controller) mountIngress(ing *api.Ingress, reload bool) error {
	projections := map[string]ioutilz.FileProjection{}
	err := c.projectIngress(ing, projections)
	if err != nil {
		return err
	}
	changed, err := c.writer.Write(projections)
	if err != nil {
		return err
	}
	if changed && reload {
		return runCmd(c.options.CmdFile)
	}
	return nil
}
