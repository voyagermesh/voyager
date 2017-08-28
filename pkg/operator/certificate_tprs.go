package operator

import (
	"errors"
	"reflect"

	"github.com/appscode/log"
	sapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/certificate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initCertificateTPRWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.ExtClient.Certificates(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.ExtClient.Certificates(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&sapi.Certificate{},
		op.Opt.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cert, ok := obj.(*sapi.Certificate); ok {
					log.Infof("%s %s@%s added", cert.GroupVersionKind(), cert.Name, cert.Namespace)
					go analytics.Send(cert.GroupVersionKind().String(), "ADD", "success")

					err := certificate.NewController(op.KubeConfig, op.KubeClient, op.ExtClient, op.Opt, cert).Process()
					if err != nil {
						log.Error(err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldCert, ok := old.(*sapi.Certificate)
				if !ok {
					log.Errorln(errors.New("Invalid Certificate object"))
					return
				}
				newCert, ok := new.(*sapi.Certificate)
				if !ok {
					log.Errorln(errors.New("Invalid Certificate object"))
					return
				}
				if !reflect.DeepEqual(oldCert.Spec, newCert.Spec) {
					err := certificate.NewController(op.KubeConfig, op.KubeClient, op.ExtClient, op.Opt, newCert).Process()
					if err != nil {
						log.Error(err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if cert, ok := obj.(*sapi.Certificate); ok {
					log.Infof("%s %s@%s deleted", cert.GroupVersionKind(), cert.Name, cert.Namespace)

					go analytics.Send(cert.GroupVersionKind().String(), "DELETE", "success")
				}
			},
		},
	)
	return informer
}
