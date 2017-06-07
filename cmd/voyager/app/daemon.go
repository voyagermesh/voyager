package app

import (
	"fmt"
	"os"
	"time"

	"github.com/appscode/go/runtime"
	"github.com/appscode/log"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/cmd/voyager/app/options"
	"github.com/appscode/voyager/pkg/analytics"
	acw "github.com/appscode/voyager/pkg/watcher"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

func Run(config *options.Config) {
	log.Infoln("Configuration:", config)
	defer runtime.HandleCrash()

	if config.EnableAnalytics {
		analytics.Enable()
	}

	c, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	w := &Watcher{
		Watcher: acw.Watcher{
			Client:     clientset.NewForConfigOrDie(c),
			ExtClient:  acs.NewForConfigOrDie(c),
			SyncPeriod: time.Minute * 2,
		},
		ProviderName:      config.ProviderName,
		ClusterName:       config.ClusterName,
		LoadbalancerImage: config.LoadbalancerImageName,
		IngressClass:      config.IngressClass,
	}

	log.Infoln("configuration loadded, running watcher")
	go w.Run()

	analytics.VoyagerStarted()
}
