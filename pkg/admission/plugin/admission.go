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

package plugin

import (
	"encoding/json"
	"fmt"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
)

type CRDValidator struct {
	CloudProvider string
}

var _ hooks.AdmissionHook = &CRDValidator{}

func (a *CRDValidator) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.voyager.appscode.com",
			Version:  "v1beta1",
			Resource: "admissionreviews",
		},
		"admissionreview"
}

func (a *CRDValidator) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	supportedKinds := sets.NewString(api.ResourceKindIngress)

	if (req.Operation != admission.Create && req.Operation != admission.Update) ||
		len(req.SubResource) != 0 ||
		(req.Kind.Group != api.SchemeGroupVersion.Group || !supportedKinds.Has(req.Kind.Kind) &&
			(req.Kind.Group != "extensions" || req.Kind.Kind != "Ingress")) {
		status.Allowed = true
		return status
	}

	switch req.Kind.Kind {
	case api.ResourceKindIngress:
		obj := &api.Ingress{}
		err := json.Unmarshal(req.Object.Raw, obj)
		if err != nil {
			return hooks.StatusBadRequest(err)
		}

		if req.Kind.Group == "extensions" {
			if obj.Annotations == nil {
				obj.Annotations = make(map[string]string)
			}
			obj.Annotations[api.APISchema] = api.APISchemaIngress
		}
		obj.Migrate()

		if req.Operation == admission.Update {
			oldObj := &api.Ingress{}
			err := json.Unmarshal(req.OldObject.Raw, oldObj)
			if err != nil {
				return hooks.StatusBadRequest(err)
			}
			if obj.LBType() != oldObj.LBType() {
				return hooks.StatusBadRequest(fmt.Errorf("can't change load balancer type from %s to %s", oldObj.LBType(), obj.LBType()))
			}
		}

		err = obj.IsValid(a.CloudProvider)
		if err != nil {
			return hooks.StatusForbidden(err)
		}
	}

	status.Allowed = true
	return status
}

func (a *CRDValidator) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
