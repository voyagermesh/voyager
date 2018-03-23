package server

import (
	"flag"
	"fmt"
	"time"

	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	"github.com/appscode/kutil/meta"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/appscode/voyager/pkg/admission/plugin"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/haproxy/template"
	"github.com/appscode/voyager/pkg/operator"
	prom "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/pkg/errors"
	"github.com/prometheus/haproxy_exporter/collector"
	"github.com/spf13/pflag"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
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
	QPS                         float64
	Burst                       int
	ResyncPeriod                time.Duration
	MaxNumRequeues              int
	NumThreads                  int
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string

	PrometheusCrdGroup string
	PrometheusCrdKinds prom.CrdKinds

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
		HAProxyImageTag:   "1.8.4-6.0.0",
		ExporterImageTag:  "6.0.0",
		OperatorNamespace: meta.Namespace(),
		OperatorService:   "voyager-operator",
		EnableRBAC:        false,
		ResyncPeriod:      10 * time.Minute,
		MaxNumRequeues:    5,
		NumThreads:        2,
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst:              1e6,
		PrometheusCrdGroup: prom.Group,
		PrometheusCrdKinds: prom.DefaultCrdKinds,

		customTemplates:           "",
		OpsAddress:                fmt.Sprintf(":%d", api.DefaultExporterPortNumber),
		haProxyServerMetricFields: collector.ServerMetrics.String(),
		haProxyTimeout:            5 * time.Second,
	}
}

func (s *OperatorOptions) AddGoFlags(fs *flag.FlagSet) {
	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	fs.StringVar(&s.CloudProvider, "cloud-provider", s.CloudProvider, "Name of cloud provider")
	fs.StringVar(&s.CloudConfigFile, "cloud-config", s.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	fs.StringVar(&s.IngressClass, "ingress-class", s.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	fs.BoolVar(&s.EnableRBAC, "rbac", s.EnableRBAC, "Enable RBAC for operator & offshoot Kubernetes objects")
	fs.StringVar(&s.customTemplates, "custom-templates", s.customTemplates, "Glob pattern of custom HAProxy template files used to override built-in templates")

	fs.StringVar(&s.DockerRegistry, "docker-registry", s.DockerRegistry, "Docker image registry for HAProxy and Prometheus exporter")
	fs.StringVar(&s.HAProxyImageTag, "haproxy-image-tag", s.HAProxyImageTag, "Tag of Docker image containing HAProxy binary")
	fs.StringVar(&s.ExporterImageTag, "exporter-image-tag", s.ExporterImageTag, "Tag of Docker image containing Prometheus exporter")

	fs.StringVar(&s.OperatorService, "operator-service", s.OperatorService, "Name of service used to expose voyager operator")
	fs.BoolVar(&s.RestrictToOperatorNamespace, "restrict-to-operator-namespace", s.RestrictToOperatorNamespace, "If true, voyager operator will only handle Kubernetes objects in its own namespace.")

	fs.StringVar(&s.OpsAddress, "ops-address", s.OpsAddress, "Address to listen on for web interface and telemetry.")
	fs.StringVar(&s.haProxyServerMetricFields, "haproxy.server-metric-fields", s.haProxyServerMetricFields, "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
	fs.DurationVar(&s.haProxyTimeout, "haproxy.timeout", s.haProxyTimeout, "Timeout for trying to get stats from HAProxy.")

	fs.StringVar(&s.PrometheusCrdGroup, "prometheus-crd-apigroup", s.PrometheusCrdGroup, "prometheus CRD  API group name")
	fs.Var(&s.PrometheusCrdKinds, "prometheus-crd-kinds", " - EXPERIMENTAL (could be removed in future releases) - customize CRD kind names")
}

func (s *OperatorOptions) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("voyager", flag.ExitOnError)
	s.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (s *OperatorOptions) ApplyTo(cfg *operator.OperatorConfig) error {
	var err error

	err = template.LoadTemplates(config.BuiltinTemplates, s.customTemplates)
	if err != nil {
		return err
	}

	cfg.Burst = s.Burst
	cfg.CloudConfigFile = s.CloudConfigFile
	cfg.CloudProvider = s.CloudProvider
	cfg.EnableRBAC = s.EnableRBAC
	cfg.ExporterImage = s.ExporterImage()
	cfg.HAProxyImage = s.HAProxyImage()
	cfg.IngressClass = s.IngressClass
	cfg.MaxNumRequeues = s.MaxNumRequeues
	cfg.NumThreads = s.NumThreads
	cfg.OperatorNamespace = s.OperatorNamespace
	cfg.OperatorService = s.OperatorService
	cfg.OpsAddress = s.OpsAddress
	cfg.QPS = float32(s.QPS)
	cfg.RestrictToOperatorNamespace = s.RestrictToOperatorNamespace
	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.WatchNamespace = s.WatchNamespace()

	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst

	if cfg.KubeClient, err = kubernetes.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.VoyagerClient, err = cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.CRDClient, err = kext_cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.PromClient, err = prom.NewForConfig(&s.PrometheusCrdKinds, s.PrometheusCrdGroup, cfg.ClientConfig); err != nil {
		return err
	}

	cfg.AdmissionHooks = []hooks.AdmissionHook{&plugin.CRDValidator{
		CloudProvider: s.CloudProvider,
	}}

	cfg.OpsAddress = s.OpsAddress

	return nil
}

func (s *OperatorOptions) Validate() []error {
	var errs []error
	if s.HAProxyImageTag == "" {
		errs = append(errs, errors.Errorf("missing required flag --haproxy-image-tag"))
	}
	if s.CloudProvider == "$VOYAGER_CLOUD_PROVIDER" {
		errs = append(errs, errors.Errorf("invalid cloud provider `--cloud-provider=$VOYAGER_CLOUD_PROVIDER`"))
	}
	if s.CloudConfigFile == "$CLOUD_CONFIG" {
		errs = append(errs, errors.Errorf("invalid cloud config file `--cloud-config=$CLOUD_CONFIG`"))
	}
	if s.IngressClass == "$INGRESS_CLASS" {
		errs = append(errs, errors.Errorf("invalid ingress class `--ingress-class=$INGRESS_CLASS`"))
	}
	return errs
}
