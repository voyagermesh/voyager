package cmds

import (
	"fmt"
	"io/ioutil"

	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/operator"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdCheck() *cobra.Command {
	var (
		ktx      string
		fromFile string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Ingress",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile != "" {
				ingBytes, err := ioutil.ReadFile(fromFile)
				if err != nil {
					return err
				}

				var ing api.Ingress
				err = yaml.Unmarshal(ingBytes, &ing)
				if err != nil {
					return err
				}
				err = ing.IsValid(opt.CloudProvider)
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
			}

			// Detect Config from local kubeconfig
			rules := clientcmd.NewDefaultClientConfigLoadingRules()
			rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
			overrides := &clientcmd.ConfigOverrides{
				CurrentContext:  ktx,
				ClusterDefaults: clientcmd.ClusterDefaults,
			}
			config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
			if err != nil {
				return err
			}

			kubeClient = clientset.NewForConfigOrDie(config)
			extClient = acs.NewForConfigOrDie(config)
			promClient, err := pcm.NewForConfig(config)
			if err != nil {
				return err
			}

			w := operator.New(config, kubeClient, extClient, promClient, opt)
			// https://github.com/appscode/voyager/issues/346
			err = w.ValidateIngress()
			if err != nil {
				return err
			}
			fmt.Println("No validation error was found.")
			return nil
		},
	}
	cmd.Flags().StringVar(&ktx, "kube-context", ktx, "Name of Kubeconfig context")
	cmd.Flags().StringVar(&fromFile, "from-file", fromFile, "YAML formatted file containing ingress")
	cmd.Flags().StringVarP(&opt.CloudProvider, "cloud-provider", "c", opt.CloudProvider, "Name of cloud provider")
	cmd.Flags().StringVar(&opt.IngressClass, "ingress-class", opt.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	return cmd
}
