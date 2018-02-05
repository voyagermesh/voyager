package cmds

import (
	"os"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	cs "github.com/appscode/voyager/client"
	hpc "github.com/appscode/voyager/pkg/haproxy/controller"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	opts = hpc.Options{
		IngressRef: core.ObjectReference{
			Namespace: meta.Namespace(),
		},
		ConfigDir: "/etc/haproxy",
		CertDir:   "/etc/ssl/private/haproxy",
		CmdFile:   "",
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst:          1e6,
		ResyncPeriod:   5 * time.Minute,
		MaxNumRequeues: 5,
		NumThreads:     1,
	}
	initOnly bool
)

func NewCmdHAProxyController() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "haproxy-controller [command]",
		Short:             `Synchronizes HAProxy config`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			runTLSMounter()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().Float32Var(&opt.QPS, "qps", opt.QPS, "The maximum QPS to the master from this client")
	cmd.Flags().IntVar(&opt.Burst, "burst", opt.Burst, "The maximum burst for throttle")

	cmd.Flags().StringVar(&opts.IngressRef.APIVersion, "ingress-api-version", opts.IngressRef.APIVersion, "API version of ingress resource")
	cmd.Flags().StringVar(&opts.IngressRef.Name, "ingress-name", opts.IngressRef.Name, "Name of ingress resource")
	cmd.Flags().StringVar(&opts.CertDir, "cert-dir", opts.CertDir, "Path where tls certificates are stored for HAProxy")
	cmd.Flags().StringVarP(&opts.CmdFile, "reload-cmd", "b", opts.CmdFile, "Bash script that will be run to reload HAProxy")
	cmd.Flags().DurationVar(&opts.ResyncPeriod, "resync-period", opts.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
	cmd.Flags().StringVarP(&opts.CloudProvider, "cloud-provider", "c", opts.CloudProvider, "Name of cloud provider")
	cmd.Flags().BoolVar(&initOnly, "init-only", initOnly, "If true, exits after initial tls mount")

	return cmd
}

func runTLSMounter() {
	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}
	config.Burst = opts.Burst
	config.QPS = opts.QPS

	// creates the clientset
	k8sClient := kubernetes.NewForConfigOrDie(config)
	voyagerClient := cs.NewForConfigOrDie(config)

	ctrl := hpc.New(k8sClient, voyagerClient, opts)
	if err := ctrl.Setup(); err != nil {
		log.Fatalln(err)
	}
	if initOnly {
		os.Exit(0)
	}

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go ctrl.Run(stop)

	// Wait forever
	select {}
}
