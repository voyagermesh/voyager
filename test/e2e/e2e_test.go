package e2e

import (
	"testing"

	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/log"
	"fmt"
)

func init() {
	framework.Initialize()
}

func TestE2E(t *testing.T) {
	log.Infoln("Running e2e tests")
	fmt.Println(framework.TestRealToken)
}
