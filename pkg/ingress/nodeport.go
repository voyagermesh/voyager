package ingress

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
)

type nodePortController struct {
	*controller
	CloudManager cloudprovider.Interface
}

var _ Controller = &nodePortController{}

func NewNodePortController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	c := &nodePortController{
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
	c.logger.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			c.logger.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			c.logger.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			c.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			c.logger.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			c.logger.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			c.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "acs" {
		cloudInterface, err := cloudprovider.InitCloudProvider("azure", opt.CloudConfigFile)
		if err != nil {
			c.logger.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			c.logger.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			c.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		c.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		c.logger.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return c
}

func (c *nodePortController) Create() error {
	// Service is created first so that NodePorts can be used in the haproxy.conf
	if svc, vt, err := c.ensureService(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressServiceCreateFailed,
			"Failed to ensure NodePort Service, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if err = c.EnsureFirewall(svc); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to ensure firewall, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully %s NodePort Service",
			vt,
		)
	}

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
			"Failed to ensure NodePort Pods, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully %s NodePort Pods",
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

func (c *nodePortController) EnsureFirewall(svc *core.Service) error {
	if c.CloudManager != nil {
		if fw, ok := c.CloudManager.Firewall(); ok {
			nodes, err := c.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
			if err != nil {
				return err
			}
			hostnames := make([]string, len(nodes.Items))
			for i, node := range nodes.Items {
				hostnames[i] = node.Name
			}
			err = fw.EnsureFirewall(svc, hostnames)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// make sure all delete calls require only ingress name and namespace
func (c *nodePortController) Delete() {
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
	if c.CloudManager != nil { // TODO@Dipta: fix this
		if fw, ok := c.CloudManager.Firewall(); ok {
			err := fw.EnsureFirewallDeleted(&core.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Ingress.OffshootName(),
					Namespace: c.Ingress.Namespace,
				},
			})
			if err != nil {
				c.logger.Errorln(err)
			}
		}
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

func (c *nodePortController) waitForNodePortAssignment() error {
	return wait.Poll(time.Second*5, time.Minute*5, wait.ConditionFunc(func() (bool, error) {
		svc, err := c.KubeClient.CoreV1().
			Services(c.Ingress.Namespace).
			Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, port := range svc.Spec.Ports {
			if port.NodePort <= 0 {
				return false, errors.New("Port not assigned")
			}
		}
		return true, nil
	}))
}

func (c *nodePortController) ensureService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchService(c.KubeClient, meta, func(obj *core.Service) *core.Service {
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()
		obj.ObjectMeta = c.ensureOwnerReference(obj.ObjectMeta)

		obj.Spec.Type = core.ServiceTypeNodePort
		obj.Spec.Ports = []core.ServicePort{}
		obj.Spec.Selector = c.Ingress.OffshootLabels()
		obj.Spec.ExternalIPs = c.Ingress.Spec.ExternalIPs
		// https://github.com/kubernetes/kubernetes/issues/33586
		// obj.Spec.LoadBalancerSourceRanges: lbc.Config.Spec.LoadBalancerSourceRanges,

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
			obj.Spec.Ports = append(obj.Spec.Ports, p)
		}

		if ans, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
			for k, v := range ans {
				obj.Annotations[k] = v
			}
		}

		if c.Ingress.KeepSourceIP() {
			// https://github.com/appscode/voyager/issues/276
			// ref: https://kubernetes.io/docs/tasks/services/source-ip/#source-ip-for-services-with-typeloadbalancer
			obj.Spec.ExternalTrafficPolicy = core.ServiceExternalTrafficPolicyTypeLocal
		}

		return obj
	})
}

func (c *nodePortController) ensurePods() (*apps.Deployment, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return apps_util.CreateOrPatchDeployment(c.KubeClient, meta, func(obj *apps.Deployment) *apps.Deployment {
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

func (c *nodePortController) deletePods() error {
	policy := metav1.DeletePropagationForeground
	err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil {
		return err
	}
	return c.deletePodsForSelector(&metav1.LabelSelector{MatchLabels: c.Ingress.OffshootLabels()})
}
