package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeSelector(t *testing.T) {
	dataTable := map[string]map[string]string{
		"key=value": {
			"key": "value",
		},

		"key1=value1,key2=value2": {
			"key1": "value1",
			"key2": "value2",
		},

		"name=value,foo=bar=foo,": {
			"name": "value",
			"foo":  "bar=foo",
		},
	}

	for k, v := range dataTable {
		assert.Equal(t, v, parseDaemonNodeSelector(k))
	}
}

func TestGetMap(t *testing.T) {
	in := map[string]string{
		"k1": `{"o1": "v1"}`,
	}

	actual, _ := getMap(in, "k1")
	assert.Equal(t, map[string]string{"o1": "v1"}, actual)
}

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
