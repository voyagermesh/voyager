package operator

import (
	"errors"
	"reflect"
	"time"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/benbjohnson/clock"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initCertificateCRDWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.VoyagerClient.Certificates(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.VoyagerClient.Certificates(op.Opt.WatchNamespace()).Watch(metav1.ListOptions{})
		},
	}
	_, informer := cache.NewInformer(lw,
		&api.Certificate{},
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cert, ok := obj.(*api.Certificate); ok {
					ctx := etx.Background()
					log.New(ctx).Infof("%s %s@%s added", cert.GroupVersionKind(), cert.Name, cert.Namespace)
					if _, err := op.MigrateCertificate(cert); err != nil {
						op.recorder.Eventf(
							cert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateMigration,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					ctrl, err := certificate.NewController(ctx, op.KubeClient, op.VoyagerClient, op.Opt, cert)
					if err != nil {
						op.recorder.Event(
							cert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateInvalid,
							err.Error(),
						)
						return
					}
					if err := ctrl.Process(); err != nil {
						op.recorder.Event(
							cert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateInvalid,
							err.Error(),
						)
					} else {
						op.recorder.Eventf(
							cert.ObjectReference(),
							apiv1.EventTypeNormal,
							eventer.EventReasonCertificateIssueSuccessful,
							"Successfully issued certificate",
						)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				ctx := etx.Background()
				logger := log.New(ctx)
				oldCert, ok := old.(*api.Certificate)
				if !ok {
					logger.Errorln(errors.New("invalid Certificate object"))
					return
				}
				newCert, ok := new.(*api.Certificate)
				if !ok {
					logger.Errorln(errors.New("invalid Certificate object"))
					return
				}

				if !reflect.DeepEqual(oldCert.Spec, newCert.Spec) {
					if _, err := op.MigrateCertificate(newCert); err != nil {
						op.recorder.Eventf(
							newCert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateMigration,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					ctrl, err := certificate.NewController(ctx, op.KubeClient, op.VoyagerClient, op.Opt, newCert)
					if err != nil {
						op.recorder.Event(
							newCert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateIssueFailed,
							err.Error(),
						)
						return
					}
					if err := ctrl.Process(); err != nil {
						op.recorder.Event(
							newCert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateInvalid,
							err.Error(),
						)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if cert, ok := obj.(*api.Certificate); ok {
					log.New(etx.Background()).Infof("%s %s@%s deleted", cert.GroupVersionKind(), cert.Name, cert.Namespace)
				}
			},
		},
	)
	return informer
}

func (op *Operator) CheckCertificates() {
	Time := clock.New()
	ctx := etx.Background()
	for {
		select {
		case <-Time.After(time.Minute * 5):
			result, err := op.VoyagerClient.Certificates(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
			if err != nil {
				log.Error(err)
				continue
			}
			for i := range result.Items {
				cert := result.Items[i]
				if cert.IsRateLimited() {
					log.Infoln("skipping certificate %s@%s, since rate limited", cert.Name, cert.Namespace)
					continue
				}
				ctrl, err := certificate.NewController(ctx, op.KubeClient, op.VoyagerClient, op.Opt, &cert)
				if err != nil {
					op.recorder.Event(
						cert.ObjectReference(),
						apiv1.EventTypeWarning,
						eventer.EventReasonCertificateInvalid,
						err.Error(),
					)
					return
				}
				if err := ctrl.Process(); err != nil {
					op.recorder.Event(
						cert.ObjectReference(),
						apiv1.EventTypeWarning,
						eventer.EventReasonCertificateInvalid,
						err.Error(),
					)
				}
			}
		}
	}
}
