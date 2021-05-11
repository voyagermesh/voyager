/*
Copyright AppsCode Inc. and Contributors

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
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/pkg/eventer"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func (op *Operator) ValidateIngress() error {
	var invalidIngresses []string
	ingresses, err := op.KubeClient.NetworkingV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range ingresses.Items {
		engress, err := api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
		if !engress.ShouldHandleIngress(op.IngressClass) {
			klog.Warningf("Skipping ingress %s/%s, as it is not handled by Voyager.", ing.Namespace, ing.Name)
			continue
		}
		klog.Warningf("Checking ingress %s/%s", ing.Namespace, ing.Name)
		if err := engress.IsValid(op.CloudProvider); err != nil {
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

	engresses, err := op.VoyagerClient.VoyagerV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range engresses.Items {
		ing.Migrate()
		if !ing.ShouldHandleIngress(op.IngressClass) {
			klog.Warningf("Skipping ingress %s/%s, as it is not handled by Voyager.", ing.Namespace, ing.Name)
			continue
		}
		klog.Warningf("Checking ingress %s/%s", ing.Namespace, ing.Name)
		if err := ing.IsValid(op.CloudProvider); err != nil {
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
		return errors.Errorf("one or more Ingress objects are invalid: %s", strings.Join(invalidIngresses, ", "))
	}
	return nil
}
