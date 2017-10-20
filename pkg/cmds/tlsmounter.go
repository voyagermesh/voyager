package cmds

import (
	"os"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/tlsmounter"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	opts = tlsmounter.Options{
		IngressRef: apiv1.ObjectReference{
			Namespace: kutil.Namespace(),
		},
		MountPath: "/etc/ssl/private/haproxy",
		CmdFile:   "",
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst:          1e6,
		ResyncPeriod:   5 * time.Minute,
		MaxNumRequeues: 5,
	}
	initOnly bool
)

func NewCmdTLSMounter() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tls-mounter [command]",
		Short:             `Mounts TLS certificates in HAProxy pods`,
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
	cmd.Flags().StringVar(&opts.MountPath, "mount", opts.MountPath, "Path where tls certificates are stored for HAProxy")
	cmd.Flags().StringVarP(&opts.CmdFile, "boot-cmd", "b", opts.CmdFile, "Bash script that will be run on every change of the file")
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

	// creates the clientset
	k8sClient := kubernetes.NewForConfigOrDie(config)
	voyagerClient := acs.NewForConfigOrDie(config)

	ctrl := tlsmounter.New(k8sClient, voyagerClient, opts)
	if err := ctrl.Setup(); err != nil {
		log.Fatalln(err)
	}
	config.Burst = opts.Burst
	config.QPS = opts.QPS

	if initOnly {
		os.Exit(0)
	}

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go ctrl.Run(1, stop)

	// Wait forever
	select {}
}
