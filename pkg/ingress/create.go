package ingress

import (
	"fmt"
	"strings"

	"github.com/appscode/errors"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	ExporterSidecarTag = "appscode/voyager:3.1.4"
)

func (c *controller) ensureConfigMap() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := core_util.EnsureConfigMap(c.KubeClient, meta, func(obj *apiv1.ConfigMap) *apiv1.ConfigMap {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()
		obj.Data = map[string]string{
			"haproxy.cfg": c.HAProxyConfig,
		}
		return obj
	})
	return err
}

func (c *controller) ensureRBAC() error {
	log.Infoln("Creating ServiceAccount for ingress", c.Ingress.OffshootName())
	if err := c.ensureServiceAccount(); err != nil {
		return errors.FromErr(err).Err()
	}

	log.Infoln("Creating Roles for ingress", c.Ingress.OffshootName())
	if err := c.ensureRoles(); err != nil {
		return errors.FromErr(err).Err()
	}

	log.Infoln("Creating RoleBinding for ingress", c.Ingress.OffshootName())
	if err := c.ensureRoleBinding(); err != nil {
		return errors.FromErr(err).Err()
	}

	return nil
}

func (c *controller) getExporterSidecar() (*apiv1.Container, error) {
	if !c.Ingress.Stats() {
		return nil, nil // Don't add sidecar is stats is not exposed.
	}
	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		return nil, err
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		return &apiv1.Container{
			Name: "exporter",
			Args: []string{
				"export",
				fmt.Sprintf("--address=:%d", monSpec.Prometheus.Port),
				"--v=3",
			},
			Image:           ExporterSidecarTag,
			ImagePullPolicy: apiv1.PullIfNotPresent,
			Ports: []apiv1.ContainerPort{
				{
					Name:          api.ExporterPortName,
					Protocol:      apiv1.ProtocolTCP,
					ContainerPort: int32(monSpec.Prometheus.Port),
				},
			},
		}, nil
	}
	return nil, nil
}

func (c *controller) ensureStatsService() error {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.StatsServiceName(),
		Namespace: c.Ingress.Namespace,
	}

	_, err := core_util.EnsureService(c.KubeClient, meta, func(obj *apiv1.Service) *apiv1.Service {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Labels = c.Ingress.StatsLabels()

		obj.Spec.Ports = []apiv1.ServicePort{
			{
				Name:       api.StatsPortName,
				Protocol:   "TCP",
				Port:       int32(c.Ingress.StatsPort()),
				TargetPort: intstr.FromString(api.StatsPortName),
			},
		}
		obj.Spec.Selector = c.Ingress.OffshootLabels()

		monSpec, err := c.Ingress.MonitorSpec()
		if err == nil && monSpec != nil && monSpec.Prometheus != nil {
			obj.Spec.Ports = append(obj.Spec.Ports, apiv1.ServicePort{
				Name:       api.ExporterPortName,
				Protocol:   "TCP",
				Port:       int32(monSpec.Prometheus.Port),
				TargetPort: intstr.FromString(api.ExporterPortName),
			})
		}

		return obj
	})
	return err
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

func Volumes(secretNames []string) []apiv1.Volume {
	skipper := make(map[string]bool)
	vs := make([]apiv1.Volume, 0)
	for _, s := range secretNames {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, ok := skipper[s+"-secret-volume"]; ok {
			continue
		}
		skipper[s+"-secret-volume"] = true
		sVolume := apiv1.Volume{
			Name: s + "-secret-volume",
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: s,
				},
			},
		}
		vs = append(vs, sVolume)
	}
	return vs
}

func VolumeMounts(secretNames []string) []apiv1.VolumeMount {
	skipper := make(map[string]bool)
	ms := make([]apiv1.VolumeMount, 0)
	for _, s := range secretNames {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, ok := skipper[s+"-secret-volume"]; ok {
			continue
		}
		skipper[s+"-secret-volume"] = true
		sMount := apiv1.VolumeMount{
			Name:      s + "-secret-volume",
			MountPath: "/srv/haproxy/secrets/" + s,
		}
		ms = append(ms, sMount)
	}
	return ms
}
