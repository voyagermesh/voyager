package testframework

import (
	"flag"
	"sync"

	"github.com/appscode/go/flags"
	"github.com/appscode/log"
)

func init() {
	InitTestFlags()
	//errors.Handlers.Add(logginghandler.LogHandler{})
}

func Initialize() {
	InitTestFlags()
}

type TestContextType struct {
	KubeConfig string
	testConfig
}

type testConfig struct {
	Mode    string
	Verbose bool
}

var TestContext TestContextType
var once sync.Once

func RegisterFlags() {
	log.Infoln("Registering Test flags")
	flag.StringVar(&TestContext.Mode, "mode", "unit", "Running test mode")
	flag.BoolVar(&TestContext.Verbose, "verbose", false, "Run test in verbose mode")
}

func InitTestFlags() {
	once.Do(func() {
		RegisterFlags()
		registerLogLevel()
		flag.Parse()
	})
}

// Set LogLevel to Debug if no flag is provided
func registerLogLevel() {
	flag.Set("logtostderr", "true")
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(5)
}
