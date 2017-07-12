package api

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	// Add field label conversions for kinds having selectable nothing but ObjectMeta fields.
	var err error
	for _, k := range []string{ResourceKindIngress, ResourceKindCertificate} {
		kind := k // don't close over range variables
		err = scheme.AddFieldLabelConversionFunc(GroupName+"/v1", kind,
			func(label, value string) (string, string, error) {
				switch label {
				case "metadata.name", "metadata.namespace":
					return label, value, nil
				default:
					return "", "", fmt.Errorf("field label %q not supported for %q", label, kind)
				}
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewEngressFromIngress(ing interface{}) (*Ingress, error) {
	data, err := json.Marshal(ing)
	if err != nil {
		return nil, err
	}
	r := &Ingress{}
	err = json.Unmarshal(data, r)
	if err != nil {
		return nil, err
	}
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[APISchema] = APISchemaIngress
	return r, nil
}

func NewIngressFromEngress(ing interface{}) (*extensions.Ingress, error) {
	data, err := json.Marshal(ing)
	if err != nil {
		return nil, err
	}
	r := &extensions.Ingress{}
	err = json.Unmarshal(data, r)
	if err != nil {
		return nil, err
	}
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[APISchema] = APISchemaEngress
	return r, nil
}
