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
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
)

func Run(config *options.Config) {
	log.Infoln("Configuration:", config)
	defer runtime.HandleCrash()

	// ref; https://github.com/kubernetes/kubernetes/blob/ba1666fb7b946febecfc836465d22903b687118e/cmd/kube-proxy/app/server.go#L168
	// Create a Kube Client
	// define api config source
	if config.KubeConfig == "" && config.Master == "" {
		log.Warningf("Neither --kubeconfig nor --master was specified.  Using default API client.  This might not work.")
	}
	// This creates a client, first loading any specified kubeconfig
	// file, and then overriding the Master flag, if non-empty.
	c, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: config.KubeConfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: config.Master}}).ClientConfig()
	if err != nil {
		errors.Exit(err)
	}

	w := &Watcher{
		Watcher: acw.Watcher{
			Client:                  clientset.NewForConfigOrDie(c),
			AppsCodeExtensionClient: acs.NewACExtensionsForConfigOrDie(c),
			SyncPeriod:              time.Minute * 2,
		},
		ProviderName:      config.ProviderName,
		ClusterName:       config.ClusterName,
		LoadbalancerImage: config.LoadbalancerImageName,
	}

	log.Infoln("configuration loadded, running watcher")
	go w.Run()
}
