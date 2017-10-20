package ingress

import (
	"fmt"
	"reflect"

	"github.com/appscode/go/errors"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ExporterSidecarTag       = "appscode/voyager:4.0.0-rc.18"
	TLSCertificateVolumeName = "voyager-certdir"
	ErrorFilesVolumeName     = "voyager-errorfiles"
	ErrorFilesLocation       = "/srv/voyager/errorfiles"
	ErrorFilesCommand        = "errorfile"
)

func (c *controller) ensureConfigMap() error {
	cm, err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		cm = &apiv1.ConfigMap{
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
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.StatsServiceName(),
			Namespace: c.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: c.Ingress.APISchema(),
				api.OriginName:      c.Ingress.GetName(),
			},
			Labels: c.Ingress.StatsLabels(),
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name:       api.StatsPortName,
					Protocol:   "TCP",
					Port:       int32(c.Ingress.StatsPort()),
					TargetPort: intstr.FromString(api.StatsPortName),
				},
			},
			Selector: c.Ingress.OffshootLabels(),
		},
	}
	monSpec, err := c.Ingress.MonitorSpec()
	if err == nil && monSpec != nil && monSpec.Prometheus != nil {
		svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
			Name:       api.ExporterPortName,
			Protocol:   "TCP",
			Port:       int32(monSpec.Prometheus.Port),
			TargetPort: intstr.FromString(api.ExporterPortName),
		})
	}

	s, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.StatsServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		c.logger.Infof("Creating Service %s/%s", svc.Namespace, svc.Name)
		_, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return err
	} else if err != nil {
		c.logger.Errorln(err)
		return errors.FromErr(err).Err()
	}
	s.Labels = svc.Labels
	s.Annotations = svc.Annotations
	s.Spec = svc.Spec
	c.logger.Infof("Updating Service %s/%s", s.Namespace, s.Name)
	_, err = c.KubeClient.CoreV1().Services(s.Namespace).Update(s)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
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
