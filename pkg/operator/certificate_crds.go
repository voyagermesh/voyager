package operator

import (
	"context"
	"reflect"
	"time"

	etx "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/listers/voyager/voyager"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (op *Operator) initCertificateCRDWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return op.VoyagerClient.Certificates(op.Opt.WatchNamespace()).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.VoyagerClient.Certificates(op.Opt.WatchNamespace()).Watch(options)
		},
	}

	// create the workqueue
	op.certQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Certificate")

	op.certIndexer, op.certInformer = cache.NewIndexerInformer(lw, &api.Certificate{}, op.Opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cert, ok := obj.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			if err := cert.IsValid(op.Opt.CloudProvider); err != nil {
				op.recorder.Eventf(
					cert.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonCertificateInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				op.certQueue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldCert, ok := old.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			newCert, ok := new.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			if reflect.DeepEqual(oldCert.Spec, newCert.Spec) {
				return
			}
			if err := newCert.IsValid(op.Opt.CloudProvider); err != nil {
				op.recorder.Eventf(
					newCert.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonCertificateInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			if key, err := cache.MetaNamespaceKeyFunc(new); err == nil {
				op.certQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
				op.certQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	op.certLister = voyager.NewCertificateLister(op.certIndexer)
}

func (op *Operator) runCertificateWatcher() {
	for op.processNextCertificate() {
	}
}

func (op *Operator) processNextCertificate() bool {
	key, quit := op.certQueue.Get()
	if quit {
		return false
	}
	defer op.certQueue.Done(key)

	err := op.runCertificateInjector(key.(string))
	if err == nil {
		op.certQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Certificate %v. Reason: %s", key, err)

	if op.certQueue.NumRequeues(key) < op.Opt.MaxNumRequeues {
		glog.Infof("Error syncing Certificate %v: %v", key, err)
		op.certQueue.AddRateLimited(key)
		return true
	}

	op.certQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping Certificate %q out of the queue: %v", key, err)
	return true
}

func (op *Operator) runCertificateInjector(key string) error {
	obj, exists, err := op.certIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Certificate %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for Certificate %s\n", key)

		cert := obj.(*api.Certificate).DeepCopy()
		if _, err := op.MigrateCertificate(cert); err != nil {
			op.recorder.Eventf(
				cert.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonCertificateMigration,
				"Reason: %s",
				err.Error(),
			)
			return err
		}

		ctrl, err := certificate.NewController(context.Background(), op.KubeClient, op.VoyagerClient, op.Opt, cert)
		if err != nil {
			op.recorder.Event(
				cert.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonCertificateInvalid,
				err.Error(),
			)
			return err
		}
		if err := ctrl.Process(); err != nil {
			op.recorder.Event(
				cert.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonCertificateInvalid,
				err.Error(),
			)
			return err
		}

		op.recorder.Eventf(
			cert.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonCertificateIssueSuccessful,
			"Successfully issued certificate",
		)
	}
	return nil
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
						core.EventTypeWarning,
						eventer.EventReasonCertificateInvalid,
						err.Error(),
					)
					return
				}
				if err := ctrl.Process(); err != nil {
					op.recorder.Event(
						cert.ObjectReference(),
						core.EventTypeWarning,
						eventer.EventReasonCertificateInvalid,
						err.Error(),
					)
				}
			}
		}
	}
}
