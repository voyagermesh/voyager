package main

import (
	"log"

	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/voyager/pkg/cmds"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewCmdVoyager().Execute(); err != nil {
		log.Fatal(err)
	}
}
