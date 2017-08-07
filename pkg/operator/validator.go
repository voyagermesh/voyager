package operator

import (
	"fmt"
	"strings"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/eventer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (op *Operator) ValidateIngress() error {
	invalidIngresses := []string{}
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range ingresses.Items {
		engress, err := api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
		if !engress.ShouldHandleIngress(op.Opt.IngressClass) {
			continue
		}
		if err := engress.IsValid(); err != nil {
			op.recorder.Eventf(
				engress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, engress.Namespace+"/"+engress.Name)
		}
	}

	engresses, err := op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range engresses.Items {
		if !ing.ShouldHandleIngress(op.Opt.IngressClass) {
			continue
		}
		if err := ing.IsValid(); err != nil {
			op.recorder.Eventf(
				&ing,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, ing.Namespace+"/"+ing.Name)
		}
	}

	if len(invalidIngresses) > 0 {
		return fmt.Errorf("One or more Ingress objects are invalid: %s", strings.Join(invalidIngresses, ", "))
	}
	return nil
}
