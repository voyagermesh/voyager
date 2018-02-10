package plugin

import (
	"encoding/json"
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	hookapi "github.com/appscode/voyager/pkg/admission/api"
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
)

type AdmissionHookImpl struct {
	CloudProvider string
}

var _ hookapi.AdmissionHook = &AdmissionHookImpl{}

func (a *AdmissionHookImpl) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.voyager.appscode.com",
			Version:  "v1beta1",
			Resource: "admissionreviews",
		},
		"admissionreview"
}

func (a *AdmissionHookImpl) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
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
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}
		err = obj.IsValid(a.CloudProvider)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: err.Error(),
			}
			return status
		}
	case api.ResourceKindIngress:
		obj := &api.Ingress{}
		err := json.Unmarshal(req.Object.Raw, obj)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
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
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: err.Error(),
			}
			return status
		}
	}

	status.Allowed = true
	return status
}

func (a *AdmissionHookImpl) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
