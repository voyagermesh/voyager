package operator

import (
	"fmt"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op *Operator) ValidateIngress() error {
	var invalidIngresses []string
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.options.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range ingresses.Items {
		engress, err := api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
		if !engress.ShouldHandleIngress(op.options.IngressClass) {
			log.Warningf("Skipping ingress %s/%s, as it is not handled by Voyager.", ing.Namespace, ing.Name)
			continue
		}
		log.Warningf("Checking ingress %s/%s", ing.Namespace, ing.Name)
		if err := engress.IsValid(op.options.CloudProvider); err != nil {
			op.recorder.Eventf(
				engress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, engress.Name+"@"+engress.Namespace)
		}
	}

	engresses, err := op.VoyagerClient.VoyagerV1beta1().Ingresses(op.options.WatchNamespace()).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range engresses.Items {
		ing.Migrate()
		if !ing.ShouldHandleIngress(op.options.IngressClass) {
			log.Warningf("Skipping ingress %s/%s, as it is not handled by Voyager.", ing.Namespace, ing.Name)
			continue
		}
		log.Warningf("Checking ingress %s/%s", ing.Namespace, ing.Name)
		if err := ing.IsValid(op.options.CloudProvider); err != nil {
			op.recorder.Eventf(
				ing.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, ing.Name+"@"+ing.Namespace)
		}
	}

	if len(invalidIngresses) > 0 {
		return fmt.Errorf("one or more Ingress objects are invalid: %s", strings.Join(invalidIngresses, ", "))
	}
	return nil
}
