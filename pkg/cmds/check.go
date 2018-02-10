package cmds

import (
	"fmt"
	"io/ioutil"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func NewCmdCheck() *cobra.Command {
	var (
		fromFile      string
		cloudProvider string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Ingress",
		RunE: func(cmd *cobra.Command, args []string) error {
			ingBytes, err := ioutil.ReadFile(fromFile)
			if err != nil {
				return err
			}

			var ing api.Ingress
			err = yaml.Unmarshal(ingBytes, &ing)
			if err != nil {
				return err
			}
			ing.Migrate()
			err = ing.IsValid(cloudProvider)
			if err != nil {
				return err
			}
			fmt.Println("No validation error was found.")

			bi, err := yaml.Marshal(ing)
			if err != nil {
				return err
			}
			fmt.Println(string(bi))
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", fromFile, "YAML formatted file containing ingress")
	cmd.Flags().StringVarP(&cloudProvider, "cloud-provider", "c", cloudProvider, "Name of cloud provider")
	return cmd
}
