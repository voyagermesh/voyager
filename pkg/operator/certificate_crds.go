package operator

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/go/log"
	sapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	"github.com/appscode/voyager/pkg/eventer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initCertificateCRDWatcher() cache.Controller {
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
		op.Opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cert, ok := obj.(*sapi.Certificate); ok {
					log.Infof("%s %s@%s added", cert.GroupVersionKind(), cert.Name, cert.Namespace)
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
					if err := op.IsCertificateValid(cert); err != nil {
						op.recorder.Eventf(
							cert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateInvalid,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					err := certificate.NewController(op.KubeClient, op.ExtClient, op.Opt, cert).Process()
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldCert, ok := old.(*sapi.Certificate)
				if !ok {
					log.Errorln(errors.New("invalid Certificate object"))
					return
				}
				newCert, ok := new.(*sapi.Certificate)
				if !ok {
					log.Errorln(errors.New("invalid Certificate object"))
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
					if err := op.IsCertificateValid(newCert); err != nil {
						op.recorder.Eventf(
							newCert.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonCertificateInvalid,
							"Reason: %s",
							err.Error(),
						)
						return
					}
					err := certificate.NewController(op.KubeClient, op.ExtClient, op.Opt, newCert).Process()
					if err != nil {
						log.Errorln(err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if cert, ok := obj.(*sapi.Certificate); ok {
					log.Infof("%s %s@%s deleted", cert.GroupVersionKind(), cert.Name, cert.Namespace)
				}
			},
		},
	)
	return informer
}

// IsCertificateValid is an overloaded function that will call kube apis
// with information provided in certificate spec, IsValid can not have
// references for clients, its causes import loops
func (op *Operator) IsCertificateValid(c *sapi.Certificate) error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.Spec.ChallengeProvider.HTTP != nil {
		switch c.Spec.ChallengeProvider.HTTP.Ingress.APIVersion {
		case sapi.SchemeGroupVersion.String():
			var err error
			_, err = op.ExtClient.Ingresses(c.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
				Get(c.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		case "extensions/v1beta1":
			ing, err := op.KubeClient.ExtensionsV1beta1().Ingresses(c.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
				Get(c.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			_, err = sapi.NewEngressFromIngress(ing)
			if err != nil {
				return err
			}
		default:
			return errors.New("ingress API Schema unrecognized")
		}
	}

	secret, err := op.KubeClient.CoreV1().Secrets(c.Namespace).Get(c.Spec.ACMEUserSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if _, ok := secret.Data[sapi.ACMEUserEmail]; !ok {
		return fmt.Errorf("no acme user email is provided")
	}
	return nil
}
