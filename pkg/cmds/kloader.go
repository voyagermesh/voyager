package cmds

import (
	kloader "github.com/appscode/kloader/cmds"
	"github.com/spf13/cobra"
)

func NewCmdKloader() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kloader",
		Short: "Reloads HAProxy when configmap changes",
	}
	rootCmd.AddCommand(kloader.NewCheckCmd())
	rootCmd.AddCommand(kloader.NewRunCmd())

	return rootCmd
}
