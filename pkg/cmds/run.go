package cmds

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/appscode/go/log"
	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/pat"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/haproxy"
	"github.com/appscode/voyager/pkg/migrator"
	"github.com/appscode/voyager/pkg/operator"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL      string
	kubeconfigPath string
	kubeClient     kubernetes.Interface
	extClient      cs.VoyagerV1beta1Interface

	builtinTemplates          = "/srv/voyager/templates/*.cfg"
	customTemplates           = ""
	address                   = fmt.Sprintf(":%d", api.DefaultExporterPortNumber)
	haProxyServerMetricFields = hpe.ServerMetrics.String()
	haProxyTimeout            = 5 * time.Second
	opt                       = config.Options{
		HAProxyImage:         "appscode/haproxy:1.7.9-5.0.0",
		ExporterSidecarImage: "appscode/voyager:5.0.0",
		OperatorNamespace:    meta.Namespace(),
		OperatorService:      "voyager-operator",
		EnableRBAC:           false,
		ResyncPeriod:         5 * time.Minute,
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst: 1e6,
	}
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
	cmd.Flags().Float32Var(&opt.QPS, "qps", opt.QPS, "The maximum QPS to the master from this client")
	cmd.Flags().IntVar(&opt.Burst, "burst", opt.Burst, "The maximum burst for throttle")
	cmd.Flags().DurationVar(&opt.ResyncPeriod, "resync-period", opt.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	cmd.Flags().StringVarP(&opt.CloudProvider, "cloud-provider", "c", opt.CloudProvider, "Name of cloud provider")
	cmd.Flags().StringVar(&opt.CloudConfigFile, "cloud-config", opt.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	cmd.Flags().StringVar(&opt.HAProxyImage, "haproxy-image", opt.HAProxyImage, "Docker image containing HAProxy binary")
	cmd.Flags().StringVar(&opt.ExporterSidecarImage, "exporter-sidecar-image", opt.ExporterSidecarImage, "Docker image containing Prometheus exporter")
	cmd.Flags().StringVar(&opt.IngressClass, "ingress-class", opt.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	cmd.Flags().BoolVar(&opt.EnableRBAC, "rbac", opt.EnableRBAC, "Enable RBAC for operator & offshoot Kubernetes objects")
	cmd.Flags().StringVar(&customTemplates, "custom-templates", customTemplates, "Glob pattern of custom HAProxy template files used to override built-in templates")

	cmd.Flags().StringVar(&opt.OperatorService, "operator-service", opt.OperatorService, "Name of service used to expose voyager operator")
	cmd.Flags().BoolVar(&opt.RestrictToOperatorNamespace, "restrict-to-operator-namespace", opt.RestrictToOperatorNamespace, "If true, voyager operator will only handle Kubernetes objects in its own namespace.")

	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&haProxyServerMetricFields, "haproxy.server-metric-fields", haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	cmd.Flags().DurationVar(&haProxyTimeout, "haproxy.timeout", haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	return cmd
}

func runOperator() {
	if opt.HAProxyImage == "" {
		log.Fatalln("Missing required flag --haproxy-image")
	}
	if opt.CloudProvider == "$VOYAGER_CLOUD_PROVIDER" {
		log.Fatalln("Invalid cloud provider `--cloud-provider=$VOYAGER_CLOUD_PROVIDER`")
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
	config.Burst = opts.Burst
	config.QPS = opts.QPS

	kubeClient = kubernetes.NewForConfigOrDie(config)
	var crdClient kext_cs.ApiextensionsV1beta1Interface = kext_cs.NewForConfigOrDie(config)
	extClient = cs.NewForConfigOrDie(config)
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
