package ingress

import (
	"reflect"
	"strconv"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/monitor"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type loadBalancerController struct {
	*controller
}

var _ Controller = &loadBalancerController{}

func NewLoadBalancerController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	services core.ServiceLister,
	endpoints core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	return &loadBalancerController{
		controller: &controller{
			KubeClient:      kubeClient,
			ExtClient:       extClient,
			PromClient:      promClient,
			ServiceLister:   services,
			EndpointsLister: endpoints,
			Opt:             opt,
			Ingress:         ingress,
			recorder:        eventer.NewEventRecorder(kubeClient, "voyager operator"),
		},
	}
}

func (c *loadBalancerController) IsExists() bool {
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

func (c *loadBalancerController) Create() error {
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

	time.Sleep(time.Second * 5)

	// If RBAC is enabled we need to ensure service account
	if c.Opt.EnableRBAC {
		if err := c.ensureRBAC(); err != nil {
			return err
		}
	}

	// deleteResidualPods is a safety checking deletion of previous version RC
	// This should Ignore error.
	c.deleteResidualPods()
	err = c.createNodePortPods()
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
		"Successfully created NodePortPods",
	)

	time.Sleep(time.Second * 10)
	err = c.createLoadBalancerSvc()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressServiceCreateFailed,
			"Failed to create LoadBalancerService, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressServiceCreateSuccessful,
		"Successfully created LoadBalancerService",
	)

	go c.updateStatus()

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

func (c *loadBalancerController) Update(mode UpdateMode) error {
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

	if mode&UpdateFirewall > 0 ||
		mode&RestartHAProxy > 0 ||
		mode&UpdateStats > 0 {
		// Ignore Error.
		c.deleteResidualPods()
		err := c.deleteNodePortPods()
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressUpdateFailed,
				"Failed to update Pods, %s", err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
		err = c.createNodePortPods()
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
	}

	if mode&UpdateFirewall > 0 {
		_, err := c.updateLBSvc()
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
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceUpdateSuccessful,
			"Successfully updated LBService",
		)

		go c.updateStatus()
	}

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

func (c *loadBalancerController) UpdateTargetAnnotations(old *api.Ingress, new *api.Ingress) error {
	// Check for changes in ingress.appscode.com/annotations-service
	c.updateServiceAnnotations(old, new)

	// Check for changes in ingress.appscode.com/annotations-pod
	if newPodAns, newOk := new.PodsAnnotations(); newOk {
		if oldPodAns, oldOk := old.PodsAnnotations(); oldOk {
			if !reflect.DeepEqual(oldPodAns, newPodAns) {

				dep, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
				if err != nil {
					return errors.FromErr(err).Err()
				}
				dep.Spec.Template.Annotations = mergeAnnotations(dep.Spec.Template.Annotations, oldPodAns, newPodAns)
				_, err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Update(dep)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}
	return nil
}

func (c *loadBalancerController) Delete() error {
	// Ignore Error.
	c.deleteResidualPods()
	err := c.deleteNodePortPods()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = c.deleteConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if c.Opt.EnableRBAC {
		if err := c.ensureRBACDeleted(); err != nil {
			return err
		}
	}
	err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		log.Errorln(err)
	}
	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.PromClient)
		ctrl.DeleteMonitor(c.Ingress, monSpec)
	}
	if c.Ingress.Stats() {
		c.ensureStatsServiceDeleted()
	}
	return nil
}

func (c *loadBalancerController) createLoadBalancerSvc() error {
	log.Infoln("creating LoadBalancer type lb")
	// creating service as typeLoadBalancer
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: c.Ingress.APISchema(),
				api.OriginName:      c.Ingress.GetName(),
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:                     apiv1.ServiceTypeLoadBalancer,
			Ports:                    []apiv1.ServicePort{},
			Selector:                 c.Ingress.OffshootLabels(),
			LoadBalancerSourceRanges: c.Ingress.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	mappings, err := c.Ingress.PortMappings(c.Opt.CloudProvider)
	if err != nil {
		return err
	}
	for svcPort, target := range mappings {
		p := apiv1.ServicePort{
			Name:       "tcp-" + strconv.Itoa(svcPort),
			Protocol:   "TCP",
			Port:       int32(svcPort),
			TargetPort: intstr.FromInt(target.PodPort),
			NodePort:   int32(target.NodePort),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	switch c.Opt.CloudProvider {
	case "gce", "gke":
		if ip := c.Ingress.LoadBalancerIP(); ip != nil {
			svc.Spec.LoadBalancerIP = ip.String()
		}
	}

	s, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureResourceAnnotations(s.Annotations); ok {
		s.Annotations = val
		needsUpdate = true
	}

	if isServicePortChanged(s.Spec.Ports, svc.Spec.Ports) {
		needsUpdate = true
		s.Spec.Ports = svc.Spec.Ports
	}

	if needsUpdate {
		_, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(s)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (c *loadBalancerController) createNodePortPods() error {
	log.Infoln("creating NodePort deployment")
	secrets := c.Ingress.Secrets()
	deployment := &extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Labels:    c.Ingress.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: c.Ingress.APISchema(),
				api.OriginName:      c.Ingress.GetName(),
			},
		},

		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(c.Ingress.Replicas()),
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
		},
	}

	if c.Opt.EnableRBAC {
		deployment.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
	}

	exporter, err := c.getExporterSidecar()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if exporter != nil {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, *exporter)
	}

	// adding tcp ports to pod template
	for _, podPort := range c.Ingress.PodPorts() {
		p := apiv1.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(podPort),
			Protocol:      "TCP",
			ContainerPort: int32(podPort),
		}
		deployment.Spec.Template.Spec.Containers[0].Ports = append(deployment.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if c.Ingress.Stats() {
		deployment.Spec.Template.Spec.Containers[0].Ports = append(deployment.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{
			Name:          api.StatsPortName,
			Protocol:      "TCP",
			ContainerPort: int32(c.Ingress.StatsPort()),
		})
	}

	if ans, ok := c.Ingress.PodsAnnotations(); ok {
		deployment.Spec.Template.Annotations = ans
	}

	dpl, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(deployment.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Create(deployment)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureResourceAnnotations(dpl.Annotations); ok {
		needsUpdate = true
		dpl.Annotations = val
	}

	if !reflect.DeepEqual(dpl.Spec, deployment.Spec) {
		needsUpdate = true
		dpl.Spec = deployment.Spec
	}

	if needsUpdate {
		_, err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Update(dpl)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (c *loadBalancerController) deleteNodePortPods() error {
	d, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	d.Spec.Replicas = types.Int32P(0)
	_, err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Update(d)

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

// Deprecated, creating pods using RC is now deprecated.
func (c *loadBalancerController) deleteResidualPods() error {
	rc, err := c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		log.Warningln(err)
		return err
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	rc.Spec.Replicas = types.Int32P(0)
	_, err = c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Update(rc)
	if err != nil {
		log.Warningln(err)
		return err
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		log.Warningln(err)
		return err
	}
	c.deletePodsForSelector(rc.Spec.Selector)
	return nil
}
