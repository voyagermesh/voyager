package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
