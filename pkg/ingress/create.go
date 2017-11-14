package ingress

import (
	"fmt"
	"reflect"

	"github.com/appscode/go/errors"
	core_util "github.com/appscode/kutil/core/v1"
	tools "github.com/appscode/kutil/tools/monitoring"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	TLSCertificateVolumeName = "voyager-certdir"
	ErrorFilesVolumeName     = "voyager-errorfiles"
	ErrorFilesLocation       = "/srv/voyager/errorfiles"
	ErrorFilesCommand        = "errorfile"
)

func (c *controller) ensureConfigMap() error {
	cm, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		cm = &core.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Ingress.OffshootName(),
				Namespace: c.Ingress.Namespace,
				Annotations: map[string]string{
					api.OriginAPISchema: c.Ingress.APISchema(),
					api.OriginName:      c.Ingress.GetName(),
				},
			},
			Data: map[string]string{
				"haproxy.cfg": c.HAProxyConfig,
			},
		}
		c.logger.Infof("Creating ConfigMap %s/%s", cm.Namespace, cm.Name)
		_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Create(cm)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureOriginAnnotations(cm.Annotations); ok {
		needsUpdate = true
		cm.Annotations = val
	}

	cmData := map[string]string{
		"haproxy.cfg": c.HAProxyConfig,
	}
	if !reflect.DeepEqual(cm.Data, cmData) {
		needsUpdate = true
		cm.Data = cmData
	}

	if needsUpdate {
		c.logger.Infof("Updating ConfigMap %s/%s", cm.Namespace, cm.Name)
		_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Update(cm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
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
			Args: []string{
				"export",
				fmt.Sprintf("--address=:%d", monSpec.Prometheus.Port),
				"--v=3",
			},
			Image:           c.Opt.ExporterSidecarImage,
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

func (c *controller) ensureStatsService() error {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.StatsServiceName(),
		Namespace: c.Ingress.Namespace,
	}

	_, err := core_util.CreateOrPatchService(c.KubeClient, meta, func(in *core.Service) *core.Service {
		in.Labels = c.Ingress.StatsLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Spec.Selector = c.Ingress.OffshootLabels()
		in.Spec.Ports = []core.ServicePort{
			{
				Name:       api.StatsPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(c.Ingress.StatsPort()),
				TargetPort: intstr.FromString(api.StatsPortName),
			},
		}
		monSpec, err := tools.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
		if err == nil && monSpec != nil && monSpec.Prometheus != nil {
			in.Spec.Ports = append(in.Spec.Ports, core.ServicePort{
				Name:       api.ExporterPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(monSpec.Prometheus.Port),
				TargetPort: intstr.FromString(api.ExporterPortName),
			})
		}
		return in
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
