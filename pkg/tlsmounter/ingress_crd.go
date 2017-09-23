package tlsmounter

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
)

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
	err := c.runIngressCRDInitializer(key.(string))
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
func (c *Controller) runIngressCRDInitializer(key string) error {
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
	}
	return nil
}
