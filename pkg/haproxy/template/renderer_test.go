package template

import (
	"testing"

	"github.com/appscode/go/log"
	"github.com/stretchr/testify/assert"
)

func TestCheckRenderedConfig(t *testing.T) {
	data := map[string]bool{
		`
global
	lua-load /tmp/auth-request.lua
listen http-in
	bind *:80
	server http-server 127.0.0.1:8080`: true, // valid config - required /tmp/auth-request.lua

		`listen http-in`: false, // invalid config - no bind

		``: false, // invalid config - no listener
	}

	for cfg, result := range data {
		err := CheckRenderedConfig(cfg)
		if err != nil {
			log.Errorln(err)
		}
		assert.Equal(t, result, err == nil)
	}
}
