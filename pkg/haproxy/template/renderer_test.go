package template

import (
	"testing"

	"github.com/appscode/go/log"
	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	data := map[string]bool{
		`
# global
# 	lua-load /etc/auth-request.lua
listen http-in
	bind *:80
	server http-server 127.1.0.1:8080`: true, // valid config - required /etc/auth-request.lua

		`listen http-in`: false, // invalid config - no bind

		``: false, // invalid config - no listener
	}

	for cfg, result := range data {
		err := ValidateConfig(cfg)
		if err != nil {
			log.Errorln(err)
		}
		assert.Equal(t, result, err == nil)
	}
}
