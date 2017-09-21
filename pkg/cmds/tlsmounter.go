package cmds

import (
	"log"
	"os"

	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/tlsmounter"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
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

	return cmd
}

func runSSLMounter() {
	c, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	config := &tlsmounter.TLSMountConfig{
		Namespace:     namespace(),
		IngressRef:    ingressRef(),
		MountLocation: sslMountLocation(),
		KubeConfig:    c,
		KubeClient:    clientset.NewForConfigOrDie(c),
		VoyagerClient: acs.NewForConfigOrDie(c),
	}
	op := tlsmounter.New(config)
	if err := op.Setup(); err != nil {
		log.Fatalln(err)
	}
	op.Run()
}

func ingressRef() apiv1.ObjectReference {
	ref := apiv1.ObjectReference{Kind: "Ingress"}
	if ref.APIVersion = os.Getenv("INGRESS_API_VERSION"); len(ref.APIVersion) == 0 {
		log.Fatalln("Ingress API Version not found")
	}
	if ref.Name = os.Getenv("INGRESS_NAME"); len(ref.Name) == 0 {
		log.Fatalln("Ingress Name not found")
	}
	ref.Namespace = namespace()
	return ref
}

func sslMountLocation() string {
	if loc := os.Getenv("INGRESS_SSL_MOUNT_LOCATION"); loc != "" {
		return loc
	}
	// Supports default
	return "/etc/ssl/private/haproxy"
}
