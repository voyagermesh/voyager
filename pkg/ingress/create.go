package ingress

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/monitor"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	ExporterSidecarTag = "appscode/voyager:3.1.2"
)

func (c *Controller) Create() error {
	err := c.ensureConfigMap()
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

	err = c.createLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}

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
	return nil
}

func (c *Controller) ensureConfigMap() error {
	log.Infoln("Creating ConfigMap for engress")
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
		_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Create(cm)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureResourceAnnotations(cm.Annotations); ok {
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
		_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Update(cm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (c *Controller) ensureRBAC() error {
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

func (c *Controller) createLB() error {
	if !c.SupportsLBType() {
		err := errors.Newf("LBType %s is unsupported for cloud provider: %s", c.Ingress.LBType(), c.Opt.CloudProvider).Err()
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressUnsupportedLBType,
			err.Error(),
		)
		return err
	}

	// Specifically Add Controller and Service Event Separately for all LBTypes.
	log.Infof("Creating Resource for ingress %s, LBType detected %s", c.Ingress.OffshootName(), c.Ingress.LBType())
	if c.Ingress.LBType() == api.LBTypeHostPort {
		err := c.createHostPortPods()
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressControllerCreateFailed,
				"Failed to create HostPortPods, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully created HostPortPods",
		)

		time.Sleep(time.Second * 10)
		err = c.createHostPortSvc()
		if err != nil {
			c.recorder.Eventf(
				c.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceCreateFailed,
				"Failed to create HostPortService, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully created HostPortService",
		)
	} else if c.Ingress.LBType() == api.LBTypeNodePort {
		err := c.createNodePortPods()
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

		time.Sleep(time.Second * 10)
		err = c.createNodePortSvc()
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
		c.recorder.Eventf(
			c.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully created NodePortService",
		)
	} else {
		// deleteResidualPods is a safety checking deletion of previous version RC
		// This should Ignore error.
		c.deleteResidualPods()
		err := c.createNodePortPods()
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

func (c *Controller) createHostPortSvc() error {
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

	// opening other tcp ports
	for svcPort, target := range c.PortMapping {
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

	updateFW := false
	s, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		updateFW = true
	} else if err != nil {
		return errors.FromErr(err).Err()
	} else {
		needsUpdate := false
		if val, ok := c.ensureResourceAnnotations(s.Annotations); ok {
			s.Annotations = val
			needsUpdate = true
		}

		if isServicePortChanged(s.Spec.Ports, svc.Spec.Ports) {
			needsUpdate = true
			s.Spec.Ports = svc.Spec.Ports

			// Port changed, need to update Firewall
			updateFW = true
		}

		if needsUpdate {
			_, err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Update(s)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}

	if updateFW && c.CloudManager != nil {
		daemonNodes, err := c.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(c.Ingress.NodeSelector()).String(),
		})
		if err != nil {
			log.Infoln("node not found with nodeSelector, cause", err)
			return errors.FromErr(err).Err()
		}
		// open up firewall
		log.Debugln("Checking cloud manager", c.CloudManager)
		if fw, ok := c.CloudManager.Firewall(); ok {
			log.Debugln("firewalls found")
			for _, node := range daemonNodes.Items {
				err = fw.EnsureFirewall(svc, node.Name)
				if err != nil {
					log.Errorln("Failed to ensure loadbalancer for node", node.Name, "cause", err)
				}
			}
			log.Debugln("getting firewalls for cloud manager failed")
		}
	}
	return nil
}

func (c *Controller) createHostPortPods() error {
	log.Infoln("Creating Daemon type lb for nodeSelector = ", c.Ingress.NodeSelector())
	secrets := c.Ingress.Secrets()
	daemon := &extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Labels:    c.Ingress.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: c.Ingress.APISchema(),
				api.OriginName:      c.Ingress.GetName(),
			},
		},

		Spec: extensions.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.Ingress.OffshootLabels(),
			},

			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: c.Ingress.OffshootLabels(),
				},
				Spec: apiv1.PodSpec{
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
					Volumes:     Volumes(secrets),
					HostNetwork: true,
				},
			},
		},
	}

	if c.Opt.EnableRBAC {
		daemon.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
	}

	exporter, err := c.getExporterSidecar()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if exporter != nil {
		daemon.Spec.Template.Spec.Containers = append(daemon.Spec.Template.Spec.Containers, *exporter)
	}

	// adding tcp ports to pod template
	podPorts := sets.NewInt()
	for _, target := range c.PortMapping {
		podPorts.Insert(target.PodPort)
	}
	for podPort := range podPorts {
		p := apiv1.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(podPort),
			Protocol:      "TCP",
			ContainerPort: int32(podPort),
			HostPort:      int32(podPort),
		}
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if c.Ingress.Stats() {
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{
			Name:          api.StatsPortName,
			Protocol:      "TCP",
			ContainerPort: int32(c.Ingress.StatsPort()),
		})
	}

	if ans, ok := c.Ingress.PodsAnnotations(); ok {
		daemon.Spec.Template.Annotations = ans
	}

	dm, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Get(daemon.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		log.Infoln("creating DaemonSets controller")
		_, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Create(daemon)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureResourceAnnotations(dm.Annotations); ok {
		needsUpdate = true
		dm.Annotations = val
	}

	if !reflect.DeepEqual(dm.Spec, daemon.Spec) {
		needsUpdate = true
		dm.Spec = daemon.Spec
	}

	if needsUpdate {
		_, err = c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Update(dm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (c *Controller) createNodePortSvc() error {
	log.Infoln("creating NodePort type lb")
	// creating service as type NodePort
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
			Type:     apiv1.ServiceTypeNodePort,
			Ports:    []apiv1.ServicePort{},
			Selector: c.Ingress.OffshootLabels(),
			// https://github.com/kubernetes/kubernetes/issues/33586
			// LoadBalancerSourceRanges: lbc.Config.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	for svcPort, target := range c.PortMapping {
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

	if c.Opt.CloudProvider == "aws" && c.Ingress.KeepSourceIP() {
		// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
		svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"] = "*"
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

func (c *Controller) createNodePortPods() error {
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
	podPorts := sets.NewInt()
	for _, target := range c.PortMapping {
		podPorts.Insert(target.PodPort)
	}
	for podPort := range podPorts {
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

func (c *Controller) createLoadBalancerSvc() error {
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
	for svcPort, target := range c.PortMapping {
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

func (c *Controller) getExporterSidecar() (*apiv1.Container, error) {
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

func (c *Controller) ensureStatsService() error {
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
		_, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return err
	} else if err != nil {
		log.Errorln(err)
		return errors.FromErr(err).Err()
	}
	s.Labels = svc.Labels
	s.Annotations = svc.Annotations
	s.Spec = svc.Spec
	_, err = c.KubeClient.CoreV1().Services(s.Namespace).Update(s)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Controller) updateStatus() error {
	var statuses []apiv1.LoadBalancerIngress

	switch c.Ingress.LBType() {
	case api.LBTypeLoadBalancer:
		for i := 0; i < 50; i++ {
			time.Sleep(time.Second * 10)
			if svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{}); err == nil {
				if len(svc.Status.LoadBalancer.Ingress) >= 1 {
					statuses = svc.Status.LoadBalancer.Ingress
					break
				}
			}
		}
		// TODO @sadlil consider adding node ip in status for hostport/nodeport mode
	}

	if len(statuses) > 0 {
		if c.Ingress.APISchema() == api.APISchemaIngress {
			ing, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).Get(c.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			ing, err := c.ExtClient.Ingresses(c.Ingress.Namespace).Get(c.Ingress.Name)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = c.ExtClient.Ingresses(c.Ingress.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	return nil
}

func (c *Controller) ensureResourceAnnotations(annotation map[string]string) (map[string]string, bool) {
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

func isServicePortChanged(oldPorts, newPorts []apiv1.ServicePort) bool {
	// Check if any port changed
	ports := make(map[int32]bool)
	for _, port := range oldPorts {
		// We only use TCP protocol so ports are unique for our scenario
		ports[port.Port] = true
	}

	for _, port := range newPorts {
		if _, ok := ports[port.Port]; !ok {
			return true
		}
	}
	return false
}
