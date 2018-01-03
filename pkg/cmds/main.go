package cmds

import (
	"flag"
	"log"
	"strings"

	v "github.com/appscode/go/version"
	"github.com/appscode/kutil/tools/analytics"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/kubernetes/fake"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewCmdVoyager(version string) *cobra.Command {
	var (
		enableAnalytics = true
	)
	rootCmd := &cobra.Command{
		Use:               "voyager [command]",
		Short:             `Voyager by Appscode - Secure Ingress Controller for Kubernetes`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					opt.AnalyticsClientID = analytics.ClientID()
					client.ClientID(opt.AnalyticsClientID)
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
				}
			}
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdExport(version))
	rootCmd.AddCommand(NewCmdTLSMounter())
	rootCmd.AddCommand(NewCmdCheck())
	rootCmd.AddCommand(NewCmdKloader())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
