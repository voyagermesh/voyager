package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, v, parseNodeSelector(k))
	}
}

func TestGetMap(t *testing.T) {
	in := map[string]string{
		"k1": `{"o1": "v1"}`,
	}

	actual, _ := getMap(in, "k1")
	assert.Equal(t, map[string]string{"o1": "v1"}, actual)
}
