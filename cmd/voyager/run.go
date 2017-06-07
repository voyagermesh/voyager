package main

import (
	_ "net/http/pprof"
	"syscall"
	"time"

	"github.com/appscode/go/runtime"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/log"
	acs "github.com/appscode/voyager/client/clientset"
	_ "github.com/appscode/voyager/client/clientset/fake"
	"github.com/appscode/voyager/pkg/analytics"
	"github.com/appscode/voyager/pkg/watcher"
	"github.com/mikespook/golib/signal"
	"github.com/spf13/cobra"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	_ "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

var (
	masterURL      string
	kubeconfigPath string

	ProviderName          string
	ClusterName           string
	LoadbalancerImageName string = "appscode/haproxy:1.7.5-1.5.5"
	IngressClass          string
	EnableAnalytics       bool = true

	kubeClient clientset.Interface
	extClient  acs.ExtensionInterface
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run kubedb operator in Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVarP(&ProviderName, "cloud-provider", "c", ProviderName, "Name of cloud provider")
	cmd.Flags().StringVarP(&ClusterName, "cluster-name", "k", ClusterName, "Name of Kubernetes cluster")
	cmd.Flags().StringVarP(&LoadbalancerImageName, "haproxy-image", "h", LoadbalancerImageName, "haproxy image name to be run")

	cmd.Flags().StringVar(&IngressClass, "ingress-class", "", "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")

	cmd.Flags().BoolVar(&EnableAnalytics, "analytics", EnableAnalytics, "Send analytical event to Google Analytics")

	return cmd
}

func run() {
	if LoadbalancerImageName == "" {
		log.Fatalln("Missing required flag --haproxy-image")
	}
	if stringz.Contains([]string{"aws", "gce", "gke", "azure"}, ProviderName) && ClusterName == "" {
		log.Fatalln("--cluster-name flag must be set when --cloud-provider={aws,gce,gke,azure}")
	}

	defer runtime.HandleCrash()

	if EnableAnalytics {
		analytics.Enable()
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	kubeClient = clientset.NewForConfigOrDie(config)
	extClient = acs.NewForConfigOrDie(config)

	w := &watcher.Watcher{
		Client:            kubeClient,
		ExtClient:         extClient,
		SyncPeriod:        time.Minute * 2,
		ProviderName:      ProviderName,
		ClusterName:       ClusterName,
		LoadbalancerImage: LoadbalancerImageName,
		IngressClass:      IngressClass,
	}

	log.Infoln("Starting Voyager Controller...")
	go w.Run()

	analytics.VoyagerStarted()

	sig := signal.New(nil)
	sig.Bind(syscall.SIGTERM, exit)
	sig.Bind(syscall.SIGINT, exit)
	sig.Wait()
}

func exit() uint {
	analytics.VoyagerStopped()
	return signal.BreakExit
}
