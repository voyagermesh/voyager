package main

import (
	_ "net/http/pprof"
	"os"

	"github.com/appscode/errors"
	err_logger "github.com/appscode/errors/h/log"
	v "github.com/appscode/go/version"
	"github.com/appscode/log"
	logs "github.com/appscode/log/golog"
	"github.com/appscode/voyager/cmd/voyager/app"
	"github.com/appscode/voyager/cmd/voyager/app/options"
	"github.com/mikespook/golib/signal"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/util/flag"
	"k8s.io/kubernetes/pkg/version/verflag"
)

var (
	Version         string
	VersionStrategy string
	Os              string
	Arch            string
	CommitHash      string
	GitBranch       string
	GitTag          string
	CommitTimestamp string
	BuildTimestamp  string
	BuildHost       string
	BuildHostOs     string
	BuildHostArch   string
)

func init() {
	v.Version.Version = Version
	v.Version.VersionStrategy = VersionStrategy
	v.Version.Os = Os
	v.Version.Arch = Arch
	v.Version.CommitHash = CommitHash
	v.Version.GitBranch = GitBranch
	v.Version.GitTag = GitTag
	v.Version.CommitTimestamp = CommitTimestamp
	v.Version.BuildTimestamp = BuildTimestamp
	v.Version.BuildHost = BuildHost
	v.Version.BuildHostOs = BuildHostOs
	v.Version.BuildHostArch = BuildHostArch
}

func main() {
	config := options.NewConfig()
	config.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()
	errors.Handlers.Add(err_logger.LogHandler{})

	verflag.PrintAndExitIfRequested()

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
