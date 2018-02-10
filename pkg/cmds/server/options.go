package server

import (
	"flag"
	"fmt"
	"time"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client"
	hookapi "github.com/appscode/voyager/pkg/admission/api"
	"github.com/appscode/voyager/pkg/admission/plugin"
	"github.com/appscode/voyager/pkg/haproxy/template"
	"github.com/appscode/voyager/pkg/operator"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/spf13/pflag"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type OperatorOptions struct {
	CloudProvider               string
	CloudConfigFile             string
	IngressClass                string
	EnableRBAC                  bool
	OperatorNamespace           string
	OperatorService             string
	RestrictToOperatorNamespace bool
	QPS                         float32
	Burst                       int
	ResyncPeriod                time.Duration
	MaxNumRequeues              int
	NumThreads                  int
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string

	PrometheusCrdGroup string
	PrometheusCrdKinds prom.CrdKinds

	builtinTemplates          string
	customTemplates           string
	OpsAddress                string
	haProxyServerMetricFields string
	haProxyTimeout            time.Duration
}

func (s OperatorOptions) HAProxyImage() string {
	return fmt.Sprintf("%s/haproxy:%s", s.DockerRegistry, s.HAProxyImageTag)
}

func (s OperatorOptions) ExporterImage() string {
	return fmt.Sprintf("%s/voyager:%s", s.DockerRegistry, s.ExporterImageTag)
}

func (s OperatorOptions) WatchNamespace() string {
	if s.RestrictToOperatorNamespace {
		return s.OperatorNamespace
	}
	return core.NamespaceAll
}

func NewOperatorOptions() *OperatorOptions {
	return &OperatorOptions{
		DockerRegistry:    "appscode",
		HAProxyImageTag:   "1.7.9-6.0.0-alpha.0",
		ExporterImageTag:  "6.0.0-alpha.0",
		OperatorNamespace: meta.Namespace(),
		OperatorService:   "voyager-operator",
		EnableRBAC:        false,
		ResyncPeriod:      5 * time.Minute,
		MaxNumRequeues:    5,
		NumThreads:        2,
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst:              1e6,
		PrometheusCrdGroup: prom.Group,
		PrometheusCrdKinds: prom.DefaultCrdKinds,

		builtinTemplates:          "/srv/voyager/templates/*.cfg",
		customTemplates:           "",
		OpsAddress:                fmt.Sprintf(":%d", api.DefaultExporterPortNumber),
		haProxyServerMetricFields: hpe.ServerMetrics.String(),
		haProxyTimeout:            5 * time.Second,
	}
}

func (s *OperatorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Float32Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	fs.StringVarP(&s.CloudProvider, "cloud-provider", "c", s.CloudProvider, "Name of cloud provider")
	fs.StringVar(&s.CloudConfigFile, "cloud-config", s.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	fs.StringVar(&s.IngressClass, "ingress-class", s.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	fs.BoolVar(&s.EnableRBAC, "rbac", s.EnableRBAC, "Enable RBAC for operator & offshoot Kubernetes objects")
	fs.StringVar(&s.customTemplates, "custom-templates", s.customTemplates, "Glob pattern of custom HAProxy template files used to override built-in templates")

	fs.StringVar(&s.DockerRegistry, "docker-registry", s.DockerRegistry, "Docker image registry for HAProxy and Prometheus exporter")
	fs.StringVar(&s.HAProxyImageTag, "haproxy-image-tag", s.HAProxyImageTag, "Tag of Docker image containing HAProxy binary")
	fs.StringVar(&s.ExporterImageTag, "exporter-image-tag", s.ExporterImageTag, "Tag of Docker image containing Prometheus exporter")

	fs.StringVar(&s.OperatorService, "operator-service", s.OperatorService, "Name of service used to expose voyager operator")
	fs.BoolVar(&s.RestrictToOperatorNamespace, "restrict-to-operator-namespace", s.RestrictToOperatorNamespace, "If true, voyager operator will only handle Kubernetes objects in its own namespace.")

	fs.StringVar(&s.OpsAddress, "address", s.OpsAddress, "Address to listen on for web interface and telemetry.")
	fs.StringVar(&s.haProxyServerMetricFields, "haproxy.server-metric-fields", s.haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	fs.DurationVar(&s.haProxyTimeout, "haproxy.timeout", s.haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	pfs := flag.NewFlagSet("prometheus", flag.ExitOnError)
	pfs.StringVar(&s.PrometheusCrdGroup, "prometheus-crd-apigroup", s.PrometheusCrdGroup, "prometheus CRD  API group name")
	pfs.Var(&s.PrometheusCrdKinds, "prometheus-crd-kinds", " - EXPERIMENTAL (could be removed in future releases) - customize CRD kind names")
	fs.AddGoFlagSet(pfs)
}

func (s *OperatorOptions) ApplyTo(config *operator.OperatorConfig) error {
	var err error

	err = template.LoadTemplates(s.builtinTemplates, s.customTemplates)
	if err != nil {
		return err
	}

	config.Burst = s.Burst
	config.CloudConfigFile = s.CloudConfigFile
	config.CloudProvider = s.CloudProvider
	config.EnableRBAC = s.EnableRBAC
	config.ExporterImage = s.ExporterImage()
	config.HAProxyImage = s.HAProxyImage()
	config.IngressClass = s.IngressClass
	config.MaxNumRequeues = s.MaxNumRequeues
	config.NumThreads = s.NumThreads
	config.OperatorNamespace = s.OperatorNamespace
	config.OperatorService = s.OperatorService
	config.OpsAddress = s.OpsAddress
	config.QPS = s.QPS
	config.RestrictToOperatorNamespace = s.RestrictToOperatorNamespace
	config.WatchNamespace = s.WatchNamespace()

	config.ClientConfig.QPS = s.QPS
	config.ClientConfig.Burst = s.Burst

	if config.KubeClient, err = kubernetes.NewForConfig(config.ClientConfig); err != nil {
		return err
	}
	if config.VoyagerClient, err = cs.NewForConfig(config.ClientConfig); err != nil {
		return err
	}
	if config.PromClient, err = prom.NewForConfig(&s.PrometheusCrdKinds, s.PrometheusCrdGroup, config.ClientConfig); err != nil {
		return err
	}

	config.AdmissionHooks = []hookapi.AdmissionHook{&plugin.CRDValidator{
		CloudProvider: s.CloudProvider,
	}}

	config.OpsAddress = s.OpsAddress

	return nil
}

func (s *OperatorOptions) Validate() []error {
	var errors []error
	if s.HAProxyImageTag == "" {
		errors = append(errors, fmt.Errorf("missing required flag --haproxy-image-tag"))
	}
	if s.CloudProvider == "$VOYAGER_CLOUD_PROVIDER" {
		errors = append(errors, fmt.Errorf("invalid cloud provider `--cloud-provider=$VOYAGER_CLOUD_PROVIDER`"))
	}
	if s.CloudConfigFile == "$CLOUD_CONFIG" {
		errors = append(errors, fmt.Errorf("invalid cloud config file `--cloud-config=$CLOUD_CONFIG`"))
	}
	if s.IngressClass == "$INGRESS_CLASS" {
		errors = append(errors, fmt.Errorf("invalid ingress class `--ingress-class=$INGRESS_CLASS`"))
	}
	return errors
}
