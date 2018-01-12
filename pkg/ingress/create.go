package ingress

import (
	"fmt"

	tools "github.com/appscode/kube-mon"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/tools/analytics"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	TLSCertificateVolumeName = "voyager-certdir"
	ErrorFilesVolumeName     = "voyager-errorfiles"
	ErrorFilesLocation       = "/srv/voyager/errorfiles"
	ErrorFilesCommand        = "errorfile"
)

func (c *controller) ensureConfigMap() (*core.ConfigMap, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchConfigMap(c.KubeClient, meta, func(obj *core.ConfigMap) *core.ConfigMap {
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()
		if obj.Data == nil {
			obj.Data = make(map[string]string)
		}
		obj.Data["haproxy.cfg"] = c.HAProxyConfig
		return obj
	})
}

func (c *controller) ensureRBAC() error {
	if err := c.ensureServiceAccount(); err != nil {
		return err
	}
	if err := c.ensureRoles(); err != nil {
		return err
	}
	if err := c.ensureRoleBinding(); err != nil {
		return err
	}
	return nil
}

func (c *controller) getExporterSidecar() (*core.Container, error) {
	if !c.Ingress.Stats() {
		return nil, nil // Don't add sidecar is stats is not exposed.
	}
	monSpec, err := tools.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
	if err != nil {
		return nil, err
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		return &core.Container{
			Name: "exporter",
			Args: append([]string{
				"export",
				fmt.Sprintf("--address=:%d", monSpec.Prometheus.Port),
				fmt.Sprintf("--analytics=%v", config.EnableAnalytics),
			}, config.LoggerOptions.ToFlags()...),
			Env: []core.EnvVar{
				{
					Name:  analytics.Key,
					Value: config.AnalyticsClientID,
				},
			},
			Image:           c.Opt.ExporterImage(),
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          api.ExporterPortName,
					Protocol:      core.ProtocolTCP,
					ContainerPort: int32(monSpec.Prometheus.Port),
				},
			},
		}, nil
	}
	return nil, nil
}

func (c *controller) ensureStatsService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.StatsServiceName(),
		Namespace: c.Ingress.Namespace,
	}

	return core_util.CreateOrPatchService(c.KubeClient, meta, func(in *core.Service) *core.Service {
		in.Labels = c.Ingress.StatsLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Spec.Selector = c.Ingress.OffshootLabels()

		desired := []core.ServicePort{
			{
				Name:       api.StatsPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(c.Ingress.StatsPort()),
				TargetPort: intstr.FromString(api.StatsPortName),
			},
		}
		monSpec, err := tools.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
		if err == nil && monSpec != nil && monSpec.Prometheus != nil {
			desired = append(desired, core.ServicePort{
				Name:       api.ExporterPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(monSpec.Prometheus.Port),
				TargetPort: intstr.FromString(api.ExporterPortName),
			})
		}
		in.Spec.Ports = core_util.MergeServicePorts(in.Spec.Ports, desired)
		return in
	})
}

func (c *controller) ensureOriginAnnotations(annotation map[string]string) (map[string]string, bool) {
	needsUpdate := false

	// Copy the given map to avoid updating the original annotations
	ret := annotation
	if ret == nil {
		ret = make(map[string]string)
	}

	if val := ret[api.OriginAPISchema]; val != c.Ingress.APISchema() {
		needsUpdate = true
		ret[api.OriginAPISchema] = c.Ingress.APISchema()
	}

	if val := ret[api.OriginName]; val != c.Ingress.GetName() {
		needsUpdate = true
		ret[api.OriginName] = c.Ingress.GetName()
	}
	return ret, needsUpdate
}
