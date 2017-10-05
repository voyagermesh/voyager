package cmds

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/appscode/go/log"
	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/kutil"
	"github.com/appscode/pat"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	haproxy "github.com/appscode/voyager/pkg/haproxy"
	"github.com/appscode/voyager/pkg/migrator"
	"github.com/appscode/voyager/pkg/operator"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL      string
	kubeconfigPath string
	opt            config.Options = config.Options{
		HAProxyImage:      "appscode/haproxy:1.7.9-4.0.0-rc.4",
		OperatorNamespace: kutil.Namespace(),
		OperatorService:   "voyager-operator",
		HTTPChallengePort: 56791,
		EnableRBAC:        false,
		ResyncPeriod:      5 * time.Minute,
	}

	builtinTemplates = "/srv/voyager/templates/*.cfg"
	customTemplates  = ""

	address                   string        = fmt.Sprintf(":%d", api.DefaultExporterPortNumber)
	haProxyServerMetricFields string        = hpe.ServerMetrics.String()
	haProxyTimeout            time.Duration = 5 * time.Second

	kubeClient clientset.Interface
	extClient  acs.VoyagerV1beta1Interface
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Run operator",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			runOperator()
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVarP(&opt.CloudProvider, "cloud-provider", "c", opt.CloudProvider, "Name of cloud provider")
	cmd.Flags().StringVar(&opt.CloudConfigFile, "cloud-config", opt.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	cmd.Flags().StringVar(&opt.HAProxyImage, "haproxy-image", opt.HAProxyImage, "haproxy image name to be run")
	cmd.Flags().StringVar(&opt.IngressClass, "ingress-class", opt.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	cmd.Flags().BoolVar(&opt.EnableRBAC, "rbac", opt.EnableRBAC, "Enable RBAC for operator & offshoot Kubernetes objects")
	cmd.Flags().DurationVar(&opt.ResyncPeriod, "resync-period", opt.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
	cmd.Flags().StringVar(&customTemplates, "custom-templates", customTemplates, "Glob pattern of custom HAProxy template files used to override built-in templates")

	cmd.Flags().StringVar(&opt.OperatorService, "operator-service", opt.OperatorService, "Name of service used to expose voyager operator")
	cmd.Flags().IntVar(&opt.HTTPChallengePort, "http-challenge-port", opt.HTTPChallengePort, "Port used to answer ACME HTTP challenge")

	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&haProxyServerMetricFields, "haproxy.server-metric-fields", haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	cmd.Flags().DurationVar(&haProxyTimeout, "haproxy.timeout", haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	return cmd
}

func runOperator() {
	if opt.HAProxyImage == "" {
		log.Fatalln("Missing required flag --haproxy-image")
	}
	if opt.CloudProvider == "$CLOUD_PROVIDER" {
		log.Fatalln("Invalid cloud provider `--cloud-provider=$CLOUD_PROVIDER`")
	}
	if opt.CloudConfigFile == "$CLOUD_CONFIG" {
		log.Fatalln("Invalid cloud config file `--cloud-config=$CLOUD_CONFIG`")
	}
	if opt.IngressClass == "$INGRESS_CLASS" {
		log.Fatalln("Invalid ingress class `--ingress-class=$INGRESS_CLASS`")
	}

	err := haproxy.LoadTemplates(builtinTemplates, customTemplates)
	if err != nil {
		log.Fatalln(err)
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	kubeClient = clientset.NewForConfigOrDie(config)
	crdClient := apiextensionsclient.NewForConfigOrDie(config)
	extClient = acs.NewForConfigOrDie(config)
	promClient, err := pcm.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	log.Infoln("Starting Voyager operator...")

	w := operator.New(kubeClient, crdClient, extClient, promClient, opt)
	err = w.Setup()
	if err != nil {
		log.Fatalln(err)
	}

	if err = migrator.NewMigrator(kubeClient, crdClient).RunMigration(); err != nil {
		log.Fatalln(err)
	}

	// https://github.com/appscode/voyager/issues/346
	err = w.ValidateIngress()
	if err != nil {
		log.Fatalln(err)
	}

	// https://github.com/appscode/voyager/pull/506
	err = w.MigrateCertificates()
	if err != nil {
		log.Fatalln("Failed certificate migrations:", err)
	}
	// https://github.com/appscode/voyager/issues/229
	w.PurgeOffshootsWithDeprecatedLabels()
	// https://github.com/appscode/voyager/issues/446
	w.PurgeOffshootsDaemonSet()
	go w.Run()

	m := pat.New()
	m.Get("/metrics", promhttp.Handler())
	pattern := fmt.Sprintf("/%s/v1beta1/namespaces/%s/ingresses/%s/metrics", PathParamAPIGroup, PathParamNamespace, PathParamName)
	log.Infof("URL pattern: %s", pattern)
	m.Get(pattern, http.HandlerFunc(ExportMetrics))
	m.Del(pattern, http.HandlerFunc(DeleteRegistry))
	http.Handle("/", m)
	log.Infoln("Listening on", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
