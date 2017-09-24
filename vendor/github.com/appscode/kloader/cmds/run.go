package cmds

import (
	"github.com/appscode/go/hold"
	"github.com/appscode/go/log"
	"github.com/appscode/kloader/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run and hold kloader",
		Run: func(cmd *cobra.Command, args []string) {
			if configMap != "" {
				mounter := controller.NewConfigMapMounter(getRestConfig(), configMap, mountDir, bashFile, resyncPeriod)
				mounter.Run()
			} else if secret != "" {
				mounter := controller.NewSecretMounter(getRestConfig(), secret, mountDir, bashFile, resyncPeriod)
				mounter.Run()
			}
			hold.Hold()
		},
	}
	addFlags(cmd)
	return cmd
}

func getRestConfig() *rest.Config {
	if configMap == "" && secret == "" {
		log.Fatalln("ConfigMap/Secret is required, but not provided")
	}

	if configMap != "" && secret != "" {
		log.Fatalln("Either ConfigMap or Secret is required, but both are provided")
	}

	if mountDir == "" {
		log.Fatalln("MountDir is required, but not provided")
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalln("Failed to create KubeConfig")
	}
	return config
}
