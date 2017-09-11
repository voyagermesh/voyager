package v1beta1

import (
	"encoding/json"

	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

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
