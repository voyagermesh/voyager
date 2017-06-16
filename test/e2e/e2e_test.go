package e2e

import (
	"testing"
	"github.com/appscode/log"
)

func TestE2E(t *testing.T) {
	log.Infoln("Running e2e tests suit")
	tests := NewE2ETestSuit()
	err := tests.Run()
	if err != nil {
		log.Errorln("E2E test suit failed with error", err.Error())
		t.Fail()
	}
}
