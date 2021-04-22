/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"
	"reflect"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/pkg/certificate"
	"voyagermesh.dev/voyager/pkg/eventer"

	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"gomodules.xyz/x/log"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
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
		if cert.Spec.Paused {
			glog.Infof("Skipping paused Certificate %s\n", key)
			return nil
		}
		ctrl, err := certificate.NewController(op.KubeClient, op.VoyagerClient, op.Config, cert, op.recorder)
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
		<-Time.After(time.Minute * 5)
		result, err := op.crtLister.List(labels.Everything())
		if err != nil {
			log.Error(err)
			continue
		}
		for i := range result {
			cert := result[i]
			if cert.Spec.Paused {
				glog.Infof("Skipping paused Certificate %s/%s", cert.Namespace, cert.Name)
				continue
			}

			if cert.IsRateLimited() {
				// get a new account and retry
				s := metav1.ObjectMeta{
					Namespace: cert.Namespace,
					Name:      cert.Spec.ACMEUserSecretName,
				}
				_, _, err := core_util.CreateOrPatchSecret(context.TODO(), op.KubeClient, s, func(in *core.Secret) *core.Secret {
					delete(in.Data, api.ACMEUserPrivatekey)
					delete(in.Data, api.ACMERegistrationData)

					return in
				}, metav1.PatchOptions{})
				if err != nil {
					op.recorder.Event(
						cert.ObjectReference(),
						core.EventTypeWarning,
						eventer.EventReasonCertificateInvalid,
						err.Error(),
					)
					continue
				}
			}

			ctrl, err := certificate.NewController(op.KubeClient, op.VoyagerClient, op.Config, cert, op.recorder)
			if err != nil {
				op.recorder.Event(
					cert.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonCertificateInvalid,
					err.Error(),
				)
				continue
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
