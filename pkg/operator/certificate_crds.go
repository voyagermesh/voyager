package operator

import (
	"reflect"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

func (op *Operator) initCertificateCRDWatcher() {
	op.crtInformer = op.voyagerInformerFactory.Voyager().V1beta1().Certificates().Informer()
	op.crtQueue = queue.New("Certificate", op.MaxNumRequeues, op.NumThreads, op.reconcileCertificate)
	op.crtInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cert, ok := obj.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			if err := cert.IsValid(op.CloudProvider); err != nil {
				op.recorder.Eventf(
					cert.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonCertificateInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			queue.Enqueue(op.crtQueue.GetQueue(), obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCert, ok := oldObj.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			newCert, ok := newObj.(*api.Certificate)
			if !ok {
				log.Errorln("Invalid Certificate object")
				return
			}
			if reflect.DeepEqual(oldCert.Spec, newCert.Spec) {
				return
			}
			if err := newCert.IsValid(op.CloudProvider); err != nil {
				op.recorder.Eventf(
					newCert.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonCertificateInvalid,
					"Reason: %s",
					err.Error(),
				)
				return
			}
			queue.Enqueue(op.crtQueue.GetQueue(), newObj)
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(op.crtQueue.GetQueue(), obj)
		},
	})
	op.crtLister = op.voyagerInformerFactory.Voyager().V1beta1().Certificates().Lister()
}

func (op *Operator) reconcileCertificate(key string) error {
	obj, exists, err := op.crtInformer.GetIndexer().GetByKey(key)
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
		ctrl, err := certificate.NewController(op.KubeClient, op.VoyagerClient, op.Config, cert)
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
	for {
		select {
		case <-Time.After(time.Minute * 5):
			result, err := op.crtLister.List(labels.Everything())
			if err != nil {
				log.Error(err)
				continue
			}
			for i := range result {
				cert := result[i]
				if cert.IsRateLimited() {
					log.Infof("skipping certificate %s/%s, since rate limited", cert.Namespace, cert.Name)
					continue
				}
				ctrl, err := certificate.NewController(op.KubeClient, op.VoyagerClient, op.Config, cert)
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
