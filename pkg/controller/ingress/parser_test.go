package ingress

import (
	"testing"

	"github.com/appscode/voyager/api"
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
		assert.Equal(t, v, api.ParseNodeSelector(k))
	}
}

func TestALPNOptions(t *testing.T) {
	dataTable := map[string][]string{
		"alpn h2,http/1.1,http/1.0": {
			"h2",
			"http/1.1",
			"http/1.0",
		},

		"": {},
	}

	for k, v := range dataTable {
		assert.Equal(t, k, parseALPNOptions(v))
	}
}
