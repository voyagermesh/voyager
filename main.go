package main

import (
	"log"
	"os"
	"runtime"

	"github.com/appscode/voyager/pkg/cmds"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"kmodules.xyz/client-go/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	if err := cmds.NewCmdVoyager().Execute(); err != nil {
		log.Fatal(err)
	}
}
