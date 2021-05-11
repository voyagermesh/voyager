/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"flag"
	"fmt"
	"time"

	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	"voyagermesh.dev/voyager/pkg/admission/plugin"
	"voyagermesh.dev/voyager/pkg/config"
	"voyagermesh.dev/voyager/pkg/haproxy/template"
	"voyagermesh.dev/voyager/pkg/operator"

	"github.com/pkg/errors"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"github.com/spf13/pflag"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"kmodules.xyz/client-go/meta"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type OperatorOptions struct {
	CloudProvider               string
	CloudConfigFile             string
	IngressClass                string
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
	HAProxyImageRepository      string
	ExporterImageTag            string
	ExporterImageRepository     string
	customTemplates             string
	ValidateHAProxyConfig       bool
	EnableValidatingWebhook     bool
	LicenseFile                 string
}

func (s OperatorOptions) HAProxyImage() string {
	return fmt.Sprintf("%s/%s:%s", s.DockerRegistry, s.HAProxyImageRepository, s.HAProxyImageTag)
}

func (s OperatorOptions) ExporterImage() string {
	return fmt.Sprintf("%s/%s:%s", s.DockerRegistry, s.ExporterImageRepository, s.ExporterImageTag)
}

func (s OperatorOptions) WatchNamespace() string {
	if s.RestrictToOperatorNamespace {
		return s.OperatorNamespace
	}
	return core.NamespaceAll
}

func NewOperatorOptions() *OperatorOptions {
	return &OperatorOptions{
		DockerRegistry:          "appscode",
		HAProxyImageTag:         "1.9.15-v13.0.0-beta.1-alpine",
		ExporterImageTag:        "v13.0.0-beta.1",
		HAProxyImageRepository:  "haproxy",
		ExporterImageRepository: "voyager",
		OperatorNamespace:       meta.Namespace(),
		OperatorService:         "voyager-operator",
		ResyncPeriod:            10 * time.Minute,
		MaxNumRequeues:          5,
		NumThreads:              2,
		// ref: https://github.com/kubernetes/ingress-nginx/blob/e4d53786e771cc6bdd55f180674b79f5b692e552/pkg/ingress/controller/launch.go#L252-L259
		// High enough QPS to fit all expected use cases. QPS=0 is not set here, because client code is overriding it.
		QPS: 1e6,
		// High enough Burst to fit all expected use cases. Burst=0 is not set here, because client code is overriding it.
		Burst: 1e6,

		customTemplates: "",

		ValidateHAProxyConfig: true,
	}
}

func (s *OperatorOptions) AddGoFlags(fs *flag.FlagSet) {
	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	fs.StringVar(&s.CloudProvider, "cloud-provider", s.CloudProvider, "Name of cloud provider")
	fs.StringVar(&s.CloudConfigFile, "cloud-config", s.CloudConfigFile, "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	fs.StringVar(&s.IngressClass, "ingress-class", s.IngressClass, "Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.")
	fs.StringVar(&s.customTemplates, "custom-templates", s.customTemplates, "Glob pattern of custom HAProxy template files used to override built-in templates")

	fs.StringVar(&s.DockerRegistry, "docker-registry", s.DockerRegistry, "Docker image registry for HAProxy and Prometheus exporter")
	fs.StringVar(&s.HAProxyImageTag, "haproxy-image-tag", s.HAProxyImageTag, "Tag of Docker image containing HAProxy binary")
	fs.StringVar(&s.HAProxyImageRepository, "haproxy-image-repository", s.HAProxyImageRepository, "Repository of Docker image containing HAProxy binary")
	fs.StringVar(&s.ExporterImageTag, "exporter-image-tag", s.ExporterImageTag, "Tag of Docker image containing Prometheus exporter")
	fs.StringVar(&s.ExporterImageRepository, "exporter-image-repository", s.ExporterImageRepository, "Repository of Docker image containing Prometheus exporter")

	fs.StringVar(&s.OperatorService, "operator-service", s.OperatorService, "Name of service used to expose voyager operator")
	fs.BoolVar(&s.RestrictToOperatorNamespace, "restrict-to-operator-namespace", s.RestrictToOperatorNamespace, "If true, voyager operator will only handle Kubernetes objects in its own namespace.")

	fs.BoolVar(&s.ValidateHAProxyConfig, "validate-haproxy-config", s.ValidateHAProxyConfig, "If true, validates generated haproxy.cfg before sending to HAProxy pods.")

	fs.BoolVar(&s.EnableValidatingWebhook, "enable-validating-webhook", s.EnableValidatingWebhook, "If true, enables validating webhooks for Voyager CRDs.")

	fs.StringVar(&s.LicenseFile, "license-file", s.LicenseFile, "Path to license file")
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
	cfg.ExporterImage = s.ExporterImage()
	cfg.HAProxyImage = s.HAProxyImage()
	cfg.IngressClass = s.IngressClass
	cfg.MaxNumRequeues = s.MaxNumRequeues
	cfg.NumThreads = s.NumThreads
	cfg.OperatorNamespace = s.OperatorNamespace
	cfg.OperatorService = s.OperatorService
	cfg.QPS = float32(s.QPS)
	cfg.RestrictToOperatorNamespace = s.RestrictToOperatorNamespace
	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.WatchNamespace = s.WatchNamespace()
	cfg.ValidateHAProxyConfig = s.ValidateHAProxyConfig
	cfg.EnableValidatingWebhook = s.EnableValidatingWebhook
	cfg.LicenseFile = s.LicenseFile

	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst

	if cfg.KubeClient, err = kubernetes.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.WorkloadClient, err = wcs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.VoyagerClient, err = cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.CRDClient, err = crd_cs.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.PromClient, err = prom.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}

	if s.EnableValidatingWebhook {
		cfg.AdmissionHooks = []hooks.AdmissionHook{&plugin.CRDValidator{
			CloudProvider: s.CloudProvider,
		}}
	}

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
