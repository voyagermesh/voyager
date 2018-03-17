package plugin

import (
	"encoding/json"

	hooks "github.com/appscode/kutil/admission/v1beta1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
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
	supportedKinds := sets.NewString(api.ResourceKindCertificate, api.ResourceKindIngress)

	if (req.Operation != admission.Create && req.Operation != admission.Update) ||
		len(req.SubResource) != 0 ||
		(req.Kind.Group != api.SchemeGroupVersion.Group || !supportedKinds.Has(req.Kind.Kind) &&
			(req.Kind.Group != "extensions" || req.Kind.Kind != "Ingress")) {
		status.Allowed = true
		return status
	}

	switch req.Kind.Kind {
	case api.ResourceKindCertificate:
		obj := &api.Certificate{}
		err := json.Unmarshal(req.Object.Raw, obj)
		if err != nil {
			return hooks.StatusBadRequest(err)
		}
		err = obj.IsValid(a.CloudProvider)
		if err != nil {
			return hooks.StatusForbidden(err)
		}
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
