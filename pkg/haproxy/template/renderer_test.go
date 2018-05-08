package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/appscode/go/log"
)

func TestCheckRenderedConfig(t *testing.T) {
	cfg := `listen http-in
        bind *:80
        server http-server 127.0.0.1:8080
`
	err := CheckRenderedConfig(cfg)
	if err != nil {
		log.Errorln(err)
	}
	assert.Equal(t, true, err == nil)

	cfg = "listen http-in"
	err = CheckRenderedConfig(cfg)
	if err != nil {
		log.Errorln(err)
	}
	assert.Equal(t, false, err == nil)
}
