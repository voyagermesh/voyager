package operator

import (
	"fmt"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op *Operator) ValidateIngress() error {
	invalidIngresses := []string{}
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range ingresses.Items {
		engress, err := api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
		if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
			log.Warningf("Skipping ingress %s@%s, as it is not handled by Voyager.", ing.Name, ing.Namespace)
			continue
		}
		log.Warningf("Checking ingress %s@%s", ing.Name, ing.Namespace)
		if err := engress.IsValid(op.Opt.CloudProvider); err != nil {
			op.recorder.Eventf(
				engress.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, engress.Name+"@"+engress.Namespace)
		}
	}

	engresses, err := op.VoyagerClient.Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range engresses.Items {
		ing.Migrate()
		if !ing.ShouldHandleIngress(op.Opt.IngressClass) {
			log.Warningf("Skipping ingress %s@%s, as it is not handled by Voyager.", ing.Name, ing.Namespace)
			continue
		}
		log.Warningf("Checking ingress %s@%s", ing.Name, ing.Namespace)
		if err := ing.IsValid(op.Opt.CloudProvider); err != nil {
			op.recorder.Eventf(
				ing.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, ing.Name+"@"+ing.Namespace)
		}
	}

	if len(invalidIngresses) > 0 {
		return fmt.Errorf("One or more Ingress objects are invalid: %s", strings.Join(invalidIngresses, ", "))
	}
	return nil
}
