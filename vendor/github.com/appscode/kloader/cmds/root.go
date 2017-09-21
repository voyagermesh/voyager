package cmds

import (
	"flag"
	golog "log"

	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd(version string) *cobra.Command {
	var (
		enableAnalytics = true
	)
	rootCmd := &cobra.Command{
		Use: "kloader",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				golog.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(NewCheckCmd())
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
