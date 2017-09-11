package main

import (
	"log"

	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/voyager/client/scheme"
	"github.com/appscode/voyager/pkg/cmds"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/pkg/api/v1"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewCmdVoyager(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}
