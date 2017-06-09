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
