package operator

import (
	"fmt"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/eventer"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (op *Operator) ValidateIngress() error {
	invalidIngresses := []string{}
	ingresses, err := op.ingLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, ing := range ingresses {
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
				core.EventTypeWarning,
				eventer.EventReasonIngressInvalid,
				"Reason: %s",
				err.Error(),
			)
			invalidIngresses = append(invalidIngresses, engress.Name+"@"+engress.Namespace)
		}
	}

	engresses, err := op.engLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, ing := range engresses {
		ing.Migrate()
		if !ing.ShouldHandleIngress(op.Opt.IngressClass) {
			log.Warningf("Skipping ingress %s@%s, as it is not handled by Voyager.", ing.Name, ing.Namespace)
			continue
		}
		log.Warningf("Checking ingress %s@%s", ing.Name, ing.Namespace)
		if err := ing.IsValid(op.Opt.CloudProvider); err != nil {
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
		return fmt.Errorf("One or more Ingress objects are invalid: %s", strings.Join(invalidIngresses, ", "))
	}
	return nil
}
