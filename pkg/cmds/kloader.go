package cmds

import (
	"flag"
	golog "log"

	kloader "github.com/appscode/kloader/cmds"
	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCmdKloader() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "kloader",
		Short: "Reloads HAProxy when configmap changes",
	}
	rootCmd.AddCommand(kloader.NewCheckCmd())
	rootCmd.AddCommand(kloader.NewRunCmd())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
