package main

import (
	"log"

	logs "github.com/appscode/log/golog"
	"github.com/appscode/voyager/pkg/cmds"
	_ "k8s.io/client-go/kubernetes/fake"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewCmdVoyager(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}
