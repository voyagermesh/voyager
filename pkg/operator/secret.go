package operator

import (
	"reflect"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initSecretWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Secrets(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Secrets(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewIndexerInformer(lw,
		&core.Secret{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if oldSecret, ok := old.(*core.Secret); ok {
					if newSecret, ok := new.(*core.Secret); ok {
						if reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
							return
						}
						ctx := etx.Background()
						logger := log.New(ctx)
						// Secret DataChanged. We need to list all Ingress and check which of
						// those ingress uses this secret as basic auth secret.
						items, err := op.listIngresses()
						if err != nil {
							log.Errorln(err)
							return
						}

						for i := range items {
							engress := &items[i]
							if engress.ShouldHandleIngress(op.Opt.IngressClass) || op.IngressServiceUsesAuthSecret(engress, newSecret) {
								if engress.UsesAuthSecret(newSecret.Namespace, newSecret.Name) {
									if key, err := cache.MetaNamespaceKeyFunc(engress); err == nil {
										op.engQueue.Add(key)
										logger.Infof("Add/Delete/Update of secret %s@%s, Ingress %s re-queued for update", newSecret.Name, newSecret.Namespace, key)
									}
								}
							}
						}
					}
				}
			},
		},
		cache.Indexers{},
	)
	return informer
}

func (op *Operator) IngressServiceUsesAuthSecret(ing *tapi.Ingress, secret *core.Secret) bool {
	svcs, err := op.ServiceLister.List(labels.Everything())
	if err != nil {
		log.Errorln(err)
		return false
	}

	for _, svc := range svcs {
		if ing.HasBackendService(svc.Name, svc.Namespace) {
			if svc.Annotations != nil {
				if svc.Annotations[tapi.AuthSecret] == secret.Name && svc.Namespace == secret.Namespace {
					return true
				}
			}
		}
	}
	return false
}
