package tlsmounter

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func (c *Controller) runCertificateWatcher() {
	for c.processNextCertificate() {
	}
}

func (c *Controller) processNextCertificate() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.cQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.cQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.runCertificateInitializer(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.cQueue.Forget(key)
		return true
	}
	log.Errorln("Failed to process Certificate %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.cQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.cQueue.AddRateLimited(key)
		return true
	}

	c.cQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) runCertificateInitializer(key string) error {
	obj, exists, err := c.cIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Certificate, so that we will see a delete for one d
		fmt.Printf("Certificate %s does not exist anymore\n", key)
	} else {
		d := obj.(*api.Certificate)
		fmt.Printf("Sync/Add/Update for Certificate %s\n", d.GetName())
	}
	return nil
}
