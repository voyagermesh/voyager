package cmds

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	configMap, secret, mountDir, bashFile string
	masterURL, kubeconfigPath             string
	resyncPeriod                          time.Duration = 5 * time.Minute
)

func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&configMap, "configmap", "c", "", "Configmap name that needs to be mount")
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "Secret name that needs to be mount")
	cmd.Flags().StringVarP(&mountDir, "mount-location", "m", "", "Volume location where the file will be mounted")
	cmd.Flags().StringVarP(&bashFile, "boot-cmd", "b", "", "Bash script that will be run on every change of the file")
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().DurationVar(&resyncPeriod, "resync-period", resyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
}
