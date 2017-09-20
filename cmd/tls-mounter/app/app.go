package app

import (
	"flag"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/cmd/tls-mounter/app/operator"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"strings"
)

var (
	masterURL      string
	kubeconfigPath string
)

func NewCmdTLSMount() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "tls-mount [command]",
		Short:             `Mount required voyager secrets into the HAProxy pods`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	rootCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	return rootCmd
}

func run() {
	c, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	config := &operator.TLSMountConfig{
		Namespace:     namespace(),
		IngressRef:    ingressRef(),
		MountLocation: sslMountLocation(),
		KubeConfig:    c,
		KubeClient:    clientset.NewForConfigOrDie(c),
		VoyagerClient: acs.NewForConfigOrDie(c),
	}
	op := operator.New(config)
	if err := op.Setup(); err != nil {
		log.Fatalln(err)
	}
	op.Watch()
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

func namespace() string {
	if ns := os.Getenv("KUBE_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return apiv1.NamespaceDefault
}
