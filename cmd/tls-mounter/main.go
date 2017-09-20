package main

import (
	"log"

	logs "github.com/appscode/log/golog"
	"github.com/appscode/voyager/cmd/tls-mounter/app"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.NewCmdTLSMount().Execute(); err != nil {
		log.Fatal(err)
	}
}
