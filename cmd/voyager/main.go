package main

import (
	_ "net/http/pprof"
	"os"

	"github.com/appscode/errors"
	err_logger "github.com/appscode/errors/h/log"
	"github.com/appscode/go/flags"
	"github.com/appscode/log"
	logs "github.com/appscode/log/golog"
	"github.com/appscode/voyager/cmd/voyager/app"
	"github.com/appscode/voyager/cmd/voyager/app/options"
	"github.com/mikespook/golib/signal"
	"github.com/spf13/pflag"
	// Add fake package as a dependency to add this under vendor
	_ "github.com/appscode/k8s-addons/client/clientset/fake"
	_ "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func main() {
	config := options.NewConfig()
	config.AddFlags(pflag.CommandLine)

	flags.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()
	errors.Handlers.Add(err_logger.LogHandler{})
	flags.DumpAll()

	if config.ProviderName == "" ||
		config.ClusterName == "" ||
		config.LoadbalancerImageName == "" {
		log.Fatalln("Required flag not provided.")
	}

	log.Infoln("Starting Voyager Controller...")
	go app.Run(config)
	sig := signal.New(nil)
	sig.Bind(os.Interrupt, func() uint { return signal.BreakExit })
	sig.Wait()
}
