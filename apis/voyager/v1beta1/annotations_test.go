package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetTimeOuts(t *testing.T) {
	ing := &Ingress{}
	timeouts := ing.Timeouts()

	assert.Equal(t, timeouts, timeoutDefaults)
}

func TestGetOptions(t *testing.T) {
	ing := &Ingress{
		ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
			"ingress.appscode.com/default-option": `{"positive-options": "true", "negative-options": "false", "to-be-ignored": ""}`,
		}},
	}
	opt := ing.HAProxyOptions()
	responseMap := map[string]bool{
		"positive-options": true,
		"negative-options": false,
	}
	assert.Equal(t, responseMap, opt)
}
