package e2e

import (
	"testing"

	"github.com/appscode/log"
	"github.com/stretchr/testify/assert"
)

func TestE2E(t *testing.T) {
	log.Infoln("Running e2e tests suit")
	tests := NewE2ETestSuit()
	err := tests.Run()
	if !assert.Nil(t, err) {
		log.Errorln("E2E test suit failed with error", err)
	}
}
