package integration

import (
	"testing"

	"github.com/appscode/log"
	"github.com/appscode/voyager/test/e2e"
	"github.com/appscode/voyager/test/testframework"
)

func init() {
	testframework.Initialize()
}

func TestIntegration(t *testing.T) {
	log.Infoln("Integration test needs the current voyager to be deployed inside cluster")
	testsuit := e2e.NewE2ETestSuit()
	testsuit.Voyager = nil
	testsuit.Config.InCluster = true

	err := testsuit.Run()
	if err != nil {
		log.Errorln("E2E test suit failed with error", err.Error())
		t.Fail()
	}
}
