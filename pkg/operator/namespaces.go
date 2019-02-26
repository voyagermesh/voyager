package operator

import (
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/tools/queue"
)

func (op *Operator) initNamespaceWatcher() {
	op.nsInformer = op.kubeInformerFactory.Core().V1().Namespaces().Informer()
	op.nsQueue = queue.New("Namespace", op.MaxNumRequeues, op.NumThreads, op.reconcileNamespace)
	op.nsInformer.AddEventHandler(queue.NewDeleteHandler(op.nsQueue.GetQueue()))
	op.nsLister = op.kubeInformerFactory.Core().V1().Namespaces().Lister()
}

func (op *Operator) reconcileNamespace(key string) error {
	_, exists, err := op.nsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Namespace %s does not exist anymore\n", key)
		if _, name, err := cache.SplitMetaNamespaceKey(key); err != nil {
			return err
		} else {
			op.deleteCRDs(name)
		}
	}
	return nil
}

func (op *Operator) deleteCRDs(ns string) {
	if resources, err := op.VoyagerClient.VoyagerV1beta1().Certificates(ns).List(metav1.ListOptions{}); err == nil {
		for _, resource := range resources.Items {
			op.VoyagerClient.VoyagerV1beta1().Certificates(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	}
	if resources, err := op.VoyagerClient.VoyagerV1beta1().Ingresses(ns).List(metav1.ListOptions{}); err == nil {
		for _, resource := range resources.Items {
			op.VoyagerClient.VoyagerV1beta1().Ingresses(resource.Namespace).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	}
}
