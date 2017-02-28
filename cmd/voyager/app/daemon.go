package app

import (
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/runtime"
	_ "github.com/appscode/k8s-addons/api/install"
	acs "github.com/appscode/k8s-addons/client/clientset"
	acw "github.com/appscode/k8s-addons/pkg/watcher"
	"github.com/appscode/log"
	"github.com/appscode/voyager/cmd/voyager/app/options"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

func Run(config *options.Config) {
	log.Infoln("Configuration:", config)
	defer runtime.HandleCrash()

	c, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	if err != nil {
		errors.Exit(err)
	}

	w := &Watcher{
		Watcher: acw.Watcher{
			Client:                  clientset.NewForConfigOrDie(c),
			AppsCodeExtensionClient: acs.NewACExtensionsForConfigOrDie(c),
			SyncPeriod:              time.Minute * 5,
		},
		ProviderName:      config.ProviderName,
		ClusterName:       config.ClusterName,
		LoadbalancerImage: config.LoadbalancerImageName,
		IngressClass:      config.IngressClass,
	}

	log.Infoln("configuration loadded, running watcher")
	go w.Run()
}
