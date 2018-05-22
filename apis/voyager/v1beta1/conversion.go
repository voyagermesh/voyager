package v1beta1

import (
	"encoding/json"

	core "k8s.io/api/core/v1"
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

	if v, _ := get[NodeSelector](r.Annotations); v != nil {
		r.Spec.NodeSelector = v.(map[string]string)
	}
	delete(r.Annotations, NodeSelector)

	if v, _ := get[Tolerations](r.Annotations); v != nil {
		r.Spec.Tolerations = v.([]core.Toleration)
	}
	delete(r.Annotations, Tolerations)

	r.Migrate()
	return r, nil
}
