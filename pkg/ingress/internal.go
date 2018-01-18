package ingress

import (
	"context"
	"fmt"
	"strconv"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	tools "github.com/appscode/kube-mon"
	"github.com/appscode/kutil"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/tools/analytics"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
)

type internalController struct {
	*controller
}

var _ Controller = &internalController{}

func NewInternalController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	return &internalController{
		controller: &controller{
			logger:          log.New(ctx),
			KubeClient:      kubeClient,
			CRDClient:       crdClient,
			VoyagerClient:   extClient,
			PromClient:      promClient,
			ServiceLister:   serviceLister,
			EndpointsLister: endpointsLister,
			Opt:             opt,
			Ingress:         ingress,
			recorder:        eventer.NewEventRecorder(kubeClient, "voyager operator"),
		},
	}
}

func (c *internalController) Create() error {
	if err := c.generateConfig(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressHAProxyConfigCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}

	if _, vt, err := c.ensureConfigMap(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressConfigMapCreateFailed,
			"Failed to ensure ConfigMap %s, Reason: %s",
			c.Ingress.OffshootName(),
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressConfigMapCreateSuccessful,
			"Successfully %s ConfigMap %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	// If RBAC is enabled we need to ensure service account
	if c.Opt.EnableRBAC {
		err := c.ensureRBAC()
		if err != nil {
			c.recorder.Event(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressRBACFailed,
				err.Error(),
			)
			return errors.FromErr(err).Err()
		} else {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressRBACSuccessful,
				"Successfully applied RBAC",
			)
		}
	}

	if _, vt, err := c.ensurePods(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressControllerCreateFailed,
			"Failed to ensure InternalIngress Pods, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully %s InternalIngress Pods",
			vt,
		)
	}

	if _, vt, err := c.ensureService(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressServiceCreateFailed,
			"Failed to ensure InternalIngress Service, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully %s InternalIngress Service",
			vt,
		)
	}

	if c.Ingress.Stats() {
		if _, vt, err := c.ensureStatsService(); err != nil { // Error ignored intentionally
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceCreateFailed,
				"Failed to ensure StatsService. Reason: %s",
				err.Error(),
			)
		} else if vt != kutil.VerbUnchanged {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceCreateSuccessful,
				"Successfully %s StatsService %s",
				c.Ingress.StatsServiceName(),
				vt,
			)
		}
	} else {
		if err := c.ensureStatsServiceDeleted(); err != nil { // Error ignored intentionally
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceDeleteFailed,
				"Failed to delete HAProxy stats Service. Reason: %s",
				err.Error(),
			)
		} else {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceDeleteSuccessful,
				"Successfully deleted HAProxy stats Service %s",
				c.Ingress.StatsServiceName(),
			)
		}
	}

	monSpec, err := tools.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && c.Ingress.Stats() {
		if vt, err := c.ensureMonitoringAgent(monSpec); err != nil {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressServiceMonitorCreateFailed,
				err.Error(),
			)
		} else if vt != kutil.VerbUnchanged {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressServiceMonitorCreateSuccessful,
				"Successfully %s ServiceMonitor",
				vt,
			)
		}
	} else { // monitoring disabled, delete old agent, ignore error here
		if err := c.ensureMonitoringAgentDeleted(nil); err != nil {
			log.Errorf("failed to delete old monitoring agent, reason: %s", err)
		}
	}

	return nil
}

func (c *internalController) EnsureFirewall(svc *core.Service) error {
	return nil
}

// make sure all delete calls require only ingress name and namespace
func (c *internalController) Delete() {
	if err := c.deletePods(); err != nil {
		c.logger.Errorln(err)
	}
	if err := c.deleteConfigMap(); err != nil {
		c.logger.Errorln(err)
	}
	if c.Opt.EnableRBAC {
		if err := c.ensureRBACDeleted(); err != nil {
			c.logger.Errorln(err)
		}
	}
	if err := c.ensureServiceDeleted(); err != nil {
		c.logger.Errorln(err)
	}
	// delete agent before deleting stat service
	if err := c.ensureMonitoringAgentDeleted(nil); err != nil {
		c.logger.Errorln(err)
	}
	if err := c.ensureStatsServiceDeleted(); err != nil {
		c.logger.Errorln(err)
	}
	return
}

func (c *internalController) ensureService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchService(c.KubeClient, meta, func(obj *core.Service) *core.Service {
		obj.ObjectMeta = c.ensureOwnerReference(obj.ObjectMeta)
		obj.Spec.Type = core.ServiceTypeClusterIP
		obj.Spec.Selector = c.Ingress.OffshootLabels()

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		if ans, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
			for k, v := range ans {
				obj.Annotations[k] = v
			}
		}

		// ExternalIPs
		if !sets.NewString(obj.Spec.ExternalIPs...).Equal(sets.NewString(c.Ingress.Spec.ExternalIPs...)) {
			obj.Spec.ExternalIPs = c.Ingress.Spec.ExternalIPs
		}

		// store current node-port assignment
		curNodePorts := make(map[int32]int32)
		for _, p := range obj.Spec.Ports {
			curNodePorts[p.Port] = p.NodePort
		}

		obj.Spec.Ports = []core.ServicePort{}

		// opening other tcp ports
		mappings, _ := c.Ingress.PortMappings(c.Opt.CloudProvider)
		for svcPort, target := range mappings {
			p := core.ServicePort{
				Name:       "tcp-" + strconv.Itoa(svcPort),
				Protocol:   "TCP",
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt(target.PodPort),
				NodePort:   int32(target.NodePort),
			}
			if v, ok := curNodePorts[p.Port]; ok && p.NodePort == 0 {
				p.NodePort = v // avoid reassigning node-port
			}
			obj.Spec.Ports = append(obj.Spec.Ports, p)
		}

		return obj
	})
}

func (c *internalController) ensurePods() (*apps.Deployment, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return apps_util.CreateOrPatchDeployment(c.KubeClient, meta, func(obj *apps.Deployment) *apps.Deployment {
		// Annotations
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Labels = c.Ingress.OffshootLabels()
		obj.ObjectMeta = c.ensureOwnerReference(obj.ObjectMeta)
		obj.Spec.Replicas = types.Int32P(c.Ingress.Replicas())
		obj.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: c.Ingress.OffshootLabels(),
		}

		// pod template
		obj.Spec.Template = core.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: c.Ingress.OffshootLabels(),
			},
			Spec: core.PodSpec{
				Affinity:         c.Ingress.Spec.Affinity,
				SchedulerName:    c.Ingress.Spec.SchedulerName,
				Tolerations:      c.Ingress.Spec.Tolerations,
				NodeSelector:     c.Ingress.NodeSelector(),
				ImagePullSecrets: c.Ingress.Spec.ImagePullSecrets,
				Containers: []core.Container{
					{
						Name:  "haproxy",
						Image: c.Opt.HAProxyImage(),
						Args: append([]string{
							fmt.Sprintf("--analytics=%v", config.EnableAnalytics),
							fmt.Sprintf("--burst=%v", c.Opt.Burst),
							fmt.Sprintf("--cloud-provider=%s", c.Opt.CloudProvider),
							fmt.Sprintf("--ingress-api-version=%s", c.Ingress.APISchema()),
							fmt.Sprintf("--ingress-name=%s", c.Ingress.Name),
							fmt.Sprintf("--qps=%v", c.Opt.QPS),
							fmt.Sprintf("--resync-period=%v", c.Opt.ResyncPeriod),
							"--reload-cmd=/etc/sv/haproxy/reload",
						}, config.LoggerOptions.ToFlags()...),
						Env: []core.EnvVar{
							{
								Name:  analytics.Key,
								Value: config.AnalyticsClientID,
							},
						},
						Ports:     []core.ContainerPort{},
						Resources: c.Ingress.Spec.Resources,
						VolumeMounts: []core.VolumeMount{
							{
								Name:      TLSCertificateVolumeName,
								MountPath: "/etc/ssl/private/haproxy",
							},
						},
					},
				},
				Volumes: []core.Volume{
					{
						Name: TLSCertificateVolumeName,
						VolumeSource: core.VolumeSource{
							EmptyDir: &core.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}

		obj.Spec.Template.Spec.Containers[0].Env = c.ensureEnvVars(obj.Spec.Template.Spec.Containers[0].Env)
		if c.Opt.EnableRBAC {
			obj.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
		}
		if exporter, _ := c.getExporterSidecar(); exporter != nil {
			obj.Spec.Template.Spec.Containers = append(obj.Spec.Template.Spec.Containers, *exporter)
		}

		// adding tcp ports to pod template
		for _, podPort := range c.Ingress.PodPorts() {
			p := core.ContainerPort{
				Name:          "tcp-" + strconv.Itoa(podPort),
				Protocol:      "TCP",
				ContainerPort: int32(podPort),
			}
			obj.Spec.Template.Spec.Containers[0].Ports = append(obj.Spec.Template.Spec.Containers[0].Ports, p)
		}

		if c.Ingress.Stats() {
			obj.Spec.Template.Spec.Containers[0].Ports = append(obj.Spec.Template.Spec.Containers[0].Ports, core.ContainerPort{
				Name:          api.StatsPortName,
				Protocol:      "TCP",
				ContainerPort: int32(c.Ingress.StatsPort()),
			})
		}

		if ans, ok := c.Ingress.PodsAnnotations(); ok {
			obj.Spec.Template.Annotations = ans
		}

		if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
			obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				obj.Spec.Template.Spec.Containers[0].VolumeMounts,
				core.VolumeMount{
					Name:      ErrorFilesVolumeName,
					MountPath: ErrorFilesLocation,
				})
			obj.Spec.Template.Spec.Volumes = append(
				obj.Spec.Template.Spec.Volumes,
				core.Volume{
					Name: ErrorFilesVolumeName,
					VolumeSource: core.VolumeSource{
						ConfigMap: &core.ConfigMapVolumeSource{
							LocalObjectReference: core.LocalObjectReference{
								Name: c.Ingress.ErrorFilesConfigMapName(),
							},
						},
					},
				})
		}

		return obj
	})
}

func (c *internalController) deletePods() error {
	policy := metav1.DeletePropagationForeground
	err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil {
		return err
	}
	return c.deletePodsForSelector(&metav1.LabelSelector{MatchLabels: c.Ingress.OffshootLabels()})
}
