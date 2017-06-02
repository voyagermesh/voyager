package main

import (
	_ "net/http/pprof"
	"syscall"

	"github.com/appscode/errors"
	err_logger "github.com/appscode/errors/h/log"
	"github.com/appscode/go/flags"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/log"
	logs "github.com/appscode/log/golog"
	"github.com/appscode/voyager/cmd/voyager/app"
	"github.com/appscode/voyager/cmd/voyager/app/options"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/mikespook/golib/signal"
	"github.com/spf13/pflag"
	// Add fake package as a dependency to add this under vendor
	_ "github.com/appscode/voyager/client/clientset/fake"
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

	if config.LoadbalancerImageName == "" {
		log.Fatalln("Missing required flag --haproxy-image")
	}
	if stringz.Contains([]string{"aws", "gce", "gke", "azure"}, config.ProviderName) && config.ClusterName == "" {
		log.Fatalln("--cluster-name flag must be set when --cloud-provider={aws,gce,gke,azure}")
	}

	log.Infoln("Starting Voyager Controller...")
	go app.Run(config)

	sig := signal.New(nil)
	sig.Bind(syscall.SIGTERM, exit)
	sig.Bind(syscall.SIGINT, exit)
	sig.Wait()
}

func exit() uint {
	analytics.VoyagerStopped()
	return signal.BreakExit
}
