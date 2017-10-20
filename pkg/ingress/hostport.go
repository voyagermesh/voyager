package ingress

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/monitor"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
)

type hostPortController struct {
	*controller
	CloudManager cloudprovider.Interface
}

var _ Controller = &hostPortController{}

func NewHostPortController(
	ctx context.Context,
	kubeClient clientset.Interface,
	crdClient apiextensionsclient.Interface,
	extClient acs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1alpha1Interface,
	serviceLister core.ServiceLister,
	endpointsLister core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	c := &hostPortController{
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

func (c *hostPortController) IsExists() bool {
	_, err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
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
	if c.Opt.EnableRBAC {
		_, err = c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
		_, err = c.KubeClient.RbacV1beta1().Roles(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
		_, err = c.KubeClient.RbacV1beta1().RoleBindings(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}
	return true
}

func (c *hostPortController) Create() error {
	err := c.generateConfig()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
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
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressConfigMapCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress.ObjectReference(),
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressConfigMapCreateSuccessful,
		"Successfully created ConfigMap %s",
		c.Ingress.OffshootName(),
	)

	// If RBAC is enabled we need to ensure service account
	if c.Opt.EnableRBAC {
		err := c.ensureRBAC()
		if err != nil {
			c.recorder.Event(
				c.Ingress.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressRBACFailed,
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressRBACSuccessful,
			"Successfully applied RBAC",
		)
	}

	_, err = c.ensurePods(nil)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressControllerCreateFailed,
			"Failed to create HostPortPods, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress.ObjectReference(),
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressControllerCreateSuccessful,
		"Successfully created HostPortPods",
	)

	svc, err := c.ensureService(nil)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressServiceCreateFailed,
			"Failed to create HostPortService, Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	err = c.EnsureFirewall(svc)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to ensure firewall, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress.ObjectReference(),
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressServiceCreateSuccessful,
		"Successfully created HostPortService",
	)

	if c.Ingress.Stats() {
		err := c.ensureStatsService()
		// Error ignored intentionally
		if err != nil {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceCreateFailed,
				"Failed to create Stats Service. Reason: %s",
				err.Error(),
			)
		} else {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
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
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.CRDClient, c.PromClient)
		err := ctrl.AddMonitor(c.Ingress, monSpec)
		// Error Ignored intentionally
		if err != nil {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceMonitorCreateFailed,
				err.Error(),
			)
		} else {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				apiv1.EventTypeNormal,
				eventer.EventReasonIngressServiceMonitorCreateSuccessful,
				"Successfully created ServiceMonitor",
			)
		}
	}

	return nil
}

func (c *hostPortController) Update(mode UpdateMode, old *api.Ingress) error {
	err := c.generateConfig()
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
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

	// If RBAC is enabled we need to ensure service account
	if c.Opt.EnableRBAC {
		err := c.ensureRBAC()
		if err != nil {
			c.recorder.Event(
				c.Ingress.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressRBACFailed,
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressRBACSuccessful,
			"Successfully applied RBAC",
		)
	}

	_, err = c.ensurePods(old)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressUpdateFailed,
			"Failed to update Pods, %s", err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress.ObjectReference(),
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressUpdateSuccessful,
		"Successfully updated Pods",
	)

	svc, err := c.ensureService(old)
	if err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
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
			c.Ingress.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to ensure firewall, %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	c.recorder.Eventf(
		c.Ingress.ObjectReference(),
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressServiceUpdateSuccessful,
		"Successfully updated HostPort Service",
	)

	if mode&UpdateStats > 0 {
		if c.Ingress.Stats() {
			err := c.ensureStatsService()
			if err != nil {
				c.recorder.Eventf(
					c.Ingress.ObjectReference(),
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceCreateFailed,
					"Failed to create HAProxy stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress.ObjectReference(),
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
					c.Ingress.ObjectReference(),
					apiv1.EventTypeWarning,
					eventer.EventReasonIngressStatsServiceDeleteFailed,
					"Failed to delete HAProxy stats Service. Reason: %s",
					err.Error(),
				)
			} else {
				c.recorder.Eventf(
					c.Ingress.ObjectReference(),
					apiv1.EventTypeNormal,
					eventer.EventReasonIngressStatsServiceDeleteSuccessful,
					"Successfully deleted HAProxy stats Service %s",
					c.Ingress.StatsServiceName(),
				)
			}
		}
	}
	return nil
}

func (c *hostPortController) EnsureFirewall(svc *apiv1.Service) error {
	if c.CloudManager != nil {
		if fw, ok := c.CloudManager.Firewall(); ok {
			nodes, err := c.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(c.Ingress.NodeSelector()).String(),
			})
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

func (c *hostPortController) Delete() {
	err := c.deletePods()
	if err != nil {
		c.logger.Errorln(err)
	}
	err = c.deleteConfigMap()
	if err != nil {
		c.logger.Errorln(err)
	}
	if c.Opt.EnableRBAC {
		if err := c.ensureRBACDeleted(); err != nil {
			c.logger.Errorln(err)
		}
	}

	// delete service
	err = c.ensureServiceDeleted()
	if err != nil {
		c.logger.Errorln(err)
	}

	if c.CloudManager != nil {
		if fw, ok := c.CloudManager.Firewall(); ok {
			err = fw.EnsureFirewallDeleted(&apiv1.Service{
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

	if c.Ingress.Stats() {
		c.ensureStatsServiceDeleted()
	}
	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		c.logger.Errorln(err)
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.CRDClient, c.PromClient)
		ctrl.DeleteMonitor(c.Ingress, monSpec)
	}
}

func (c *hostPortController) newService() *apiv1.Service {
	// Create a Headless service without selectors
	// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
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
			Type:      apiv1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports:     []apiv1.ServicePort{},
		},
	}
	svc.ObjectMeta = c.ensureOwnerReference(svc.ObjectMeta)

	// opening other tcp ports
	mappings, _ := c.Ingress.PortMappings(c.Opt.CloudProvider)
	for svcPort, target := range mappings {
		p := apiv1.ServicePort{
			Name:       "tcp-" + strconv.Itoa(svcPort),
			Protocol:   "TCP",
			Port:       int32(svcPort),
			TargetPort: intstr.FromInt(target.PodPort),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := c.Ingress.ServiceAnnotations(c.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}
	return svc
}

func (c *hostPortController) ensureService(old *api.Ingress) (*apiv1.Service, error) {
	desired := c.newService()
	current, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(desired.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		c.logger.Infof("Creating Service %s/%s", desired.Namespace, desired.Name)
		return c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Create(desired)
	} else if err != nil {
		return nil, err
	}
	if svc, needsUpdate := c.serviceRequiresUpdate(current, desired, old); needsUpdate {
		c.logger.Infof("Updating Service %s/%s", desired.Namespace, desired.Name)
		return c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(svc)
	}
	return current, nil
}

func (c *hostPortController) newPods() *apps.Deployment {
	deployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Labels:    c.Ingress.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: c.Ingress.APISchema(),
				api.OriginName:      c.Ingress.GetName(),
			},
		},
		Spec: apps.DeploymentSpec{
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
					Affinity:      c.Ingress.Spec.Affinity,
					SchedulerName: c.Ingress.Spec.SchedulerName,
					Tolerations:   c.Ingress.Spec.Tolerations,
					NodeSelector:  c.Ingress.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: c.Opt.HAProxyImage,
							Args: []string{
								"--ingress-api-version=" + c.Ingress.APISchema(),
								"--ingress-name=" + c.Ingress.Name,
								"--cloud-provider=" + c.Opt.CloudProvider,
								"--v=3",
								fmt.Sprintf("--qps=%v", c.Opt.QPS),
								fmt.Sprintf("--burst=%v", c.Opt.Burst),
								"--boot-cmd=" + "/etc/sv/haproxy/reload",
								"--configmap=" + c.Ingress.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
							},
							Ports:     []apiv1.ContainerPort{},
							Resources: c.Ingress.Spec.Resources,
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      TLSCertificateVolumeName,
									MountPath: "/etc/ssl/private/haproxy",
								},
							},
						},
					},
					HostNetwork: true,
					DNSPolicy:   apiv1.DNSClusterFirstWithHostNet,
					Volumes: []apiv1.Volume{
						{
							Name: TLSCertificateVolumeName,
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	deployment.ObjectMeta = c.ensureOwnerReference(deployment.ObjectMeta)
	deployment.Spec.Template.Spec.Containers[0].Env = c.ensureEnvVars(deployment.Spec.Template.Spec.Containers[0].Env)

	if c.Opt.EnableRBAC {
		deployment.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
	}

	exporter, _ := c.getExporterSidecar()
	if exporter != nil {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, *exporter)
	}

	// adding tcp ports to pod template
	for _, podPort := range c.Ingress.PodPorts() {
		p := apiv1.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(podPort),
			Protocol:      "TCP",
			ContainerPort: int32(podPort),
			HostPort:      int32(podPort),
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

	if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			apiv1.VolumeMount{
				Name:      ErrorFilesVolumeName,
				MountPath: ErrorFilesLocation,
			})

		deployment.Spec.Template.Spec.Volumes = append(
			deployment.Spec.Template.Spec.Volumes,
			apiv1.Volume{
				Name: ErrorFilesVolumeName,
				VolumeSource: apiv1.VolumeSource{
					ConfigMap: &apiv1.ConfigMapVolumeSource{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: c.Ingress.ErrorFilesConfigMapName(),
						},
					},
				},
			})
	}
	return deployment
}

func (c *hostPortController) ensurePods(old *api.Ingress) (*apps.Deployment, error) {
	desired := c.newPods()
	current, err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Get(desired.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		c.logger.Infof("Creating Deployment %s/%s", desired.Namespace, desired.Name)
		return c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Create(desired)
	} else if err != nil {
		return nil, err
	}

	needsUpdate := false

	// annotations
	if current.Annotations == nil {
		current.Annotations = make(map[string]string)
	}
	oldAnn := map[string]string{}
	if old != nil {
		if a, ok := old.PodsAnnotations(); ok {
			oldAnn = a
		}
	}
	for k, v := range desired.Annotations {
		if cv, found := current.Annotations[k]; !found || cv != v {
			current.Annotations[k] = v
			needsUpdate = true
		}
		delete(oldAnn, k)
	}
	for k := range oldAnn {
		if _, ok := current.Annotations[k]; ok {
			delete(current.Annotations, k)
			needsUpdate = true
		}
	}

	if !reflect.DeepEqual(current.Spec.Selector, desired.Spec.Selector) {
		needsUpdate = true
		current.Spec.Selector = desired.Spec.Selector
	}
	if !reflect.DeepEqual(current.Spec.Template.ObjectMeta, desired.Spec.Template.ObjectMeta) {
		needsUpdate = true
		current.Spec.Template.ObjectMeta = desired.Spec.Template.ObjectMeta
	}
	if !reflect.DeepEqual(current.Spec.Template.Annotations, desired.Spec.Template.Annotations) {
		needsUpdate = true
		current.Spec.Template.Annotations = desired.Spec.Template.Annotations
	}
	if current.Spec.Template.Spec.SchedulerName != desired.Spec.Template.Spec.SchedulerName {
		needsUpdate = true
		current.Spec.Template.Spec.SchedulerName = desired.Spec.Template.Spec.SchedulerName
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.Tolerations, desired.Spec.Template.Spec.Tolerations) {
		needsUpdate = true
		current.Spec.Template.Spec.Tolerations = desired.Spec.Template.Spec.Tolerations
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.NodeSelector, desired.Spec.Template.Spec.NodeSelector) {
		needsUpdate = true
		current.Spec.Template.Spec.NodeSelector = desired.Spec.Template.Spec.NodeSelector
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.Containers, desired.Spec.Template.Spec.Containers) {
		needsUpdate = true
		current.Spec.Template.Spec.Containers = desired.Spec.Template.Spec.Containers
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		needsUpdate = true
		current.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
	}
	if current.Spec.Template.Spec.HostNetwork != desired.Spec.Template.Spec.HostNetwork {
		needsUpdate = true
		current.Spec.Template.Spec.HostNetwork = desired.Spec.Template.Spec.HostNetwork
	}
	if current.Spec.Template.Spec.ServiceAccountName != desired.Spec.Template.Spec.ServiceAccountName {
		needsUpdate = true
		current.Spec.Template.Spec.ServiceAccountName = desired.Spec.Template.Spec.ServiceAccountName
	}
	if needsUpdate {
		c.logger.Infof("Updating Deployment %s/%s", desired.Namespace, desired.Name)
		current, err = c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Update(current)
		return current, err
	}
	return current, nil
}

func (c *hostPortController) deletePods() error {
	policy := metav1.DeletePropagationForeground
	err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil {
		return err
	}
	return c.deletePodsForSelector(&metav1.LabelSelector{MatchLabels: c.Ingress.OffshootLabels()})
}
