package operator

import (
	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchDaemonSets() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.DaemonSet{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if daemon, ok := obj.(*extensions.DaemonSet); ok {
					log.Infof("DaemonSet %s@%s deleted", daemon.Name, daemon.Namespace)
					c.restoreDaemonSetIfRequired(daemon)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (w *Operator) restoreDaemonSetIfRequired(daemon *extensions.DaemonSet) error {
	if daemon.Annotations == nil {
		return nil
	}

	// deleted resource have source reference
	engress, err := w.findOrigin(daemon.ObjectMeta)
	if err != nil {
		return err
	}
	// Ingress Still exists, restore resource
	log.Infof("DaemonSet %s@%s requires restoration", daemon.Name, daemon.Namespace)
	daemon.SelfLink = ""
	daemon.ResourceVersion = ""
	// Old resource and annotations are missing so we need to add the annotations
	if daemon.Annotations == nil {
		daemon.Annotations = make(map[string]string)
	}
	daemon.Annotations[api.OriginAPISchema] = engress.APISchema()
	daemon.Annotations[api.OriginName] = engress.Name

	_, err = w.KubeClient.ExtensionsV1beta1().DaemonSets(daemon.Namespace).Create(daemon)
	return err
}
