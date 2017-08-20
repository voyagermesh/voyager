package ingress

import (
	"strconv"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/monitor"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type nodePortController struct {
	*controller
	CloudManager cloudprovider.Interface
}

var _ Controller = &nodePortController{}

func NewNodePortController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	serviceLister core.ServiceLister,
	endpointsLister core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	ctrl := &nodePortController{
		controller: &controller{
			KubeClient:      kubeClient,
			ExtClient:       extClient,
			PromClient:      promClient,
			ServiceLister:   serviceLister,
			EndpointsLister: endpointsLister,
			Opt:             opt,
			Ingress:         ingress,
			recorder:        eventer.NewEventRecorder(kubeClient, "voyager operator"),
		},
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			ctrl.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		ctrl.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return ctrl
}

func (c *nodePortController) IsExists() bool {
	_, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	return true
}

func (c *nodePortController) Create() error {
	err := c.generateConfig()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressHAProxyConfigCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	err = c.ensureConfigMap()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressConfigMapCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressConfigMapCreateSuccessful,
		"Successfully created ConfigMap %s",
		c.Ingress.OffshootName(),
	)

	// If RBAC is enabled we need to ensure service account
	if c.Opt.EnableRBAC {
		if err := c.ensureRBAC(); err != nil {
			return err
		}
	}

	_, err = c.ensurePods(nil)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressControllerCreateFailed,
			"Failed to create NodePortPods, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressControllerCreateSuccessful,
		"Successfully created NodePortPods")

	svc, err := c.ensureService(nil)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressServiceCreateFailed,
			"Failed to create NodePortService, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	err = c.EnsureFirewall(svc)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to ensure firewall, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressServiceCreateSuccessful,
		"Successfully created NodePortService",
	)

	if c.Ingress.Stats() {
		err := c.ensureStatsService()
		// Error ignored intentionally
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceCreateFailed,
				"Failed to create Stats Service. Reason: %s",
				err.Error(),
			)
		} else {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceCreateSuccessful,
				"Successfully created Stats Service %s",
				c.Ingress.StatsServiceName(),
			)
		}
	}

	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.PromClient)
		err := ctrl.AddMonitor(c.Ingress, monSpec)
		// Error Ignored intentionally
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceMonitorCreateFailed,
				err.Error(),
			)
		} else {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeNormal,
				eventer.EventReasonIngressServiceMonitorCreateSuccessful,
				"Successfully created ServiceMonitor",
			)
		}
	}

	return nil
}

func (c *nodePortController) Update(mode UpdateMode, old *api.Ingress) error {
	err := c.generateConfig()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressHAProxyConfigCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	// Update HAProxy config
	err = c.updateConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	_, err = c.ensurePods(old)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressUpdateFailed,
			"Failed to update Pods, %s", err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressUpdateSuccessful,
		"Successfully updated Pods",
	)

	svc, err := c.ensureService(old)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressServiceUpdateFailed,
			"Failed to update LBService, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}

	err = c.EnsureFirewall(svc)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to ensure firewall, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressServiceUpdateSuccessful,
		"Successfully updated NodePort Service",
	)

	if mode&UpdateStats > 0 {
		if c.Ingress.Stats() {
			err := c.ensureStatsService()
			if err != nil {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceCreateFailed,
					"Failed to create HAProxy stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceCreateSuccessful,
					"Successfully created HAProxy stats Service %s",
					c.Ingress.StatsServiceName(),
				)
			}
		} else {
			err := c.ensureStatsServiceDeleted()
			if err != nil {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceDeleteFailed,
					"Failed to delete HAProxy stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress,
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceDeleteSuccessful,
					"Successfully deleted HAProxy stats Service %s",
					c.Ingress.StatsServiceName(),
				)
			}
		}
	}

	if mode&UpdateRBAC > 0 {
		c.ensureRoles()
	}

	return nil
}

func (c *nodePortController) EnsureFirewall(svc *apiv1.Service) error {
	if c.CloudManager != nil {
		if fw, ok := c.CloudManager.Firewall(); ok {
			// Wait for all NodePorts to be assigned first.
			for i := 0; i < 50; i++ {
				time.Sleep(time.Second * 10)
				if svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{}); err == nil {
					for _, port := range svc.Spec.Ports {
						if port.NodePort <= 0 {
							break
						}
					}
				}
			}

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

func (c *nodePortController) Delete() {
	err := c.deletePods()
	if err != nil {
		log.Errorln(err)
	}
	err = c.deleteConfigMap()
	if err != nil {
		log.Errorln(err)
	}
	if c.Opt.EnableRBAC {
		if err := c.ensureRBACDeleted(); err != nil {
			log.Errorln(err)
		}
	}
	err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		log.Errorln(err)
	}
	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		log.Errorln(err)
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.PromClient)
		ctrl.DeleteMonitor(c.Ingress, monSpec)
	}
	if c.Ingress.Stats() {
		c.ensureStatsServiceDeleted()
	}
	return
}

func (c *nodePortController) ensureService(old *api.Ingress) (*apiv1.Service, error) {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	return core_util.CreateOrPatchService(c.KubeClient, meta, func(obj *apiv1.Service) *apiv1.Service {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		oldAnn := map[string]string{}
		if old != nil {
			if a, ok := old.ServiceAnnotations(c.Opt.CloudProvider); ok {
				oldAnn = a
			}
		}
		if desired, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
			for k, v := range desired {
				obj.Annotations[k] = v
				delete(oldAnn, k)
			}
		}
		for k := range oldAnn {
			delete(obj.Annotations, k)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Spec = apiv1.ServiceSpec{
			Type:     apiv1.ServiceTypeNodePort,
			Ports:    []apiv1.ServicePort{},
			Selector: c.Ingress.OffshootLabels(),
			// https://github.com/kubernetes/kubernetes/issues/33586
			// LoadBalancerSourceRanges: lbc.Config.Spec.LoadBalancerSourceRanges,
		}

		// opening other tcp ports
		mappings, _ := c.Ingress.PortMappings(c.Opt.CloudProvider)
		for svcPort, target := range mappings {
			p := apiv1.ServicePort{
				Name:       "tcp-" + strconv.Itoa(svcPort),
				Protocol:   "TCP",
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt(target.PodPort),
				NodePort:   int32(target.NodePort),
			}
			obj.Spec.Ports = append(obj.Spec.Ports, p)
		}
		return obj
	})
}

func (c *nodePortController) ensurePods(old *api.Ingress) (*extensions.Deployment, error) {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	return ext_util.CreateOrPatchDeployment(c.KubeClient, meta, func(obj *extensions.Deployment) *extensions.Deployment {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Labels = c.Ingress.OffshootLabels()

		secrets := c.Ingress.Secrets()
		obj.Spec = extensions.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.Ingress.OffshootLabels(),
			},
			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: c.Ingress.OffshootLabels(),
				},
				Spec: apiv1.PodSpec{
					Affinity: &apiv1.Affinity{
						PodAntiAffinity: &apiv1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: apiv1.PodAffinityTerm{
										TopologyKey: "kubernetes.io/hostname",
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: c.Ingress.OffshootLabels(),
										},
									},
								},
							},
						},
					},
					NodeSelector: c.Ingress.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: c.Opt.HAProxyImage,
							Env: []apiv1.EnvVar{
								{
									Name: "KUBE_NAMESPACE",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Args: []string{
								"--config-map=" + c.Ingress.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []apiv1.ContainerPort{},
							Resources:    c.Ingress.Spec.Resources,
							VolumeMounts: VolumeMounts(secrets),
						},
					},
					Volumes: Volumes(secrets),
				},
			},
		}

		// This allows using pod autoscaler
		if obj.Spec.Replicas == nil || types.Int32(obj.Spec.Replicas) <= 1 {
			obj.Spec.Replicas = types.Int32P(c.Ingress.Replicas())
		}

		if c.Opt.EnableRBAC {
			obj.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
		}

		exporter, _ := c.getExporterSidecar()
		if exporter != nil {
			obj.Spec.Template.Spec.Containers = append(obj.Spec.Template.Spec.Containers, *exporter)
		}

		// adding tcp ports to pod template
		for _, podPort := range c.Ingress.PodPorts() {
			p := apiv1.ContainerPort{
				Name:          "tcp-" + strconv.Itoa(podPort),
				Protocol:      "TCP",
				ContainerPort: int32(podPort),
			}
			obj.Spec.Template.Spec.Containers[0].Ports = append(obj.Spec.Template.Spec.Containers[0].Ports, p)
		}

		if c.Ingress.Stats() {
			obj.Spec.Template.Spec.Containers[0].Ports = append(obj.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{
				Name:          api.StatsPortName,
				Protocol:      "TCP",
				ContainerPort: int32(c.Ingress.StatsPort()),
			})
		}

		if obj.Spec.Template.Annotations != nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		oldAnn := map[string]string{}
		if old != nil {
			if a, ok := old.PodsAnnotations(); ok {
				oldAnn = a
			}
		}
		if desired, ok := c.Ingress.PodsAnnotations(); ok {
			for k, v := range desired {
				obj.Spec.Template.Annotations[k] = v
				delete(oldAnn, k)
			}
		}
		for k := range oldAnn {
			delete(obj.Spec.Template.Annotations, k)
		}

		return obj
	})
}

func (c *nodePortController) deletePods() error {
	err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: types.FalseP(),
	})
	if err != nil {
		log.Errorln(err)
	}
	return c.deletePodsForSelector(&metav1.LabelSelector{MatchLabels: c.Ingress.OffshootLabels()})
}
