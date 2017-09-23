package cmds

import (
	"log"

	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/tlsmounter"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdTLSMount() *cobra.Command {
	ctrl := tlsmounter.Controller{
		IngressRef: apiv1.ObjectReference{
			Namespace: namespace(),
		},
		MountPath: "/etc/ssl/private/haproxy",
		CmdFile:   "",
	}
	cmd := &cobra.Command{
		Use:               "tls-mounter [command]",
		Short:             `Mounts TLS certificates in HAProxy pods`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			ctrl.KubeClient = clientset.NewForConfigOrDie(c)
			ctrl.VoyagerClient = acs.NewForConfigOrDie(c)

			if ctrl.IngressRef.APIVersion == "" {
				log.Fatalln("Ingress API Version not found")
			}
			if ctrl.IngressRef.Name == "" {
				log.Fatalln("Ingress Name not found")
			}
			if err := ctrl.Setup(); err != nil {
				log.Fatalln(err)
			}
			ctrl.Run(wait.NeverStop)
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&ctrl.IngressRef.APIVersion, "ingress-api-version", ctrl.IngressRef.APIVersion, "API version of ingress resource")
	cmd.Flags().StringVar(&ctrl.IngressRef.Name, "ingress-name", ctrl.IngressRef.Name, "Name of ingress resource")
	cmd.Flags().StringVar(&ctrl.MountPath, "mount", ctrl.MountPath, "Path where tls certificates are stored for HAProxy")
	cmd.Flags().StringVarP(&ctrl.CmdFile, "boot-cmd", "b", ctrl.CmdFile, "Bash script that will be run on every change of the file")

	return cmd
}
