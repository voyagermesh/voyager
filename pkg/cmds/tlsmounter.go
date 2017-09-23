package cmds

import (
	"log"

	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/tlsmounter"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ingressRef = apiv1.ObjectReference{
		Namespace: namespace(),
	}
	mountPath = "/etc/ssl/private/haproxy"
	bashFile  = ""
)

func NewCmdTLSMount() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tls-mounter [command]",
		Short:             `Mounts TLS certificates in HAProxy pods`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			runSSLMounter()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&ingressRef.APIVersion, "ingress-api-version", ingressRef.APIVersion, "API version of ingress resource")
	cmd.Flags().StringVar(&ingressRef.Name, "ingress-name", ingressRef.Name, "Name of ingress resource")
	cmd.Flags().StringVar(&mountPath, "mount", mountPath, "Path where tls certificates are stored for HAProxy")
	cmd.Flags().StringVarP(&bashFile, "boot-cmd", "b", "", "Bash script that will be run on every change of the file")

	return cmd
}

func runSSLMounter() {
	c, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}
	if ingressRef.APIVersion == "" {
		log.Fatalln("Ingress API Version not found")
	}
	if ingressRef.Name == "" {
		log.Fatalln("Ingress Name not found")
	}

	ctrl := tlsmounter.Controller{
		KubeClient:    clientset.NewForConfigOrDie(c),
		VoyagerClient: acs.NewForConfigOrDie(c),
		IngressRef:    ingressRef,
		MountLocation: mountPath,
	}
	if err := ctrl.Setup(); err != nil {
		log.Fatalln(err)
	}
	ctrl.Run()
}
