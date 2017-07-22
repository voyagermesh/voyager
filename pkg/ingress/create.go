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
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	ExporterSidecarTag = "appscode/voyager:3.1.1"
)

func (lbc *Controller) Create() error {
	err := lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	err = lbc.ensureConfigMap()
	if err != nil {
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressConfigMapCreateFailed,
			"Reason: %s",
			err.Error(),
		)
		return errors.FromErr(err).Err()
	}
	lbc.recorder.Eventf(
		lbc.Ingress,
		apiv1.EventTypeNormal,
		eventer.EventReasonIngressConfigMapCreateSuccessful,
		"Successfully created ConfigMap %s",
		lbc.Ingress.OffshootName(),
	)

	time.Sleep(time.Second * 5)

	// If RBAC is enabled we need to ensure service account
	if lbc.Opt.EnableRBAC {
		if err := lbc.ensureRBAC(); err != nil {
			return err
		}
	}

	err = lbc.createLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if lbc.Parsed.Stats {
		err := lbc.ensureStatsService()
		// Error ignored intentionally
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceCreateFailed,
				"Failed to create Stats Service. Reason: %s",
				err.Error(),
			)
		} else {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceCreateSuccessful,
				"Successfully created Stats Service %s",
				lbc.Ingress.StatsServiceName(),
			)
		}
	}
	return nil
}

func (lbc *Controller) ensureConfigMap() error {
	log.Infoln("Creating ConfigMap for engress")
	cm, err := lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		cm = &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      lbc.Ingress.OffshootName(),
				Namespace: lbc.Ingress.Namespace,
				Annotations: map[string]string{
					api.OriginAPISchema: lbc.Ingress.APISchema(),
					api.OriginName:      lbc.Ingress.GetName(),
				},
			},
			Data: map[string]string{
				"haproxy.cfg": lbc.ConfigData,
			},
		}
		_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Create(cm)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(cm.Annotations); ok {
		needsUpdate = true
		cm.Annotations = val
	}

	cmData := map[string]string{
		"haproxy.cfg": lbc.ConfigData,
	}
	if !reflect.DeepEqual(cm.Data, cmData) {
		needsUpdate = true
		cm.Data = cmData
	}

	if needsUpdate {
		_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Update(cm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) ensureRBAC() error {
	log.Infoln("Creating ServiceAccount for ingress", lbc.Ingress.OffshootName())
	if err := lbc.ensureServiceAccount(); err != nil {
		return errors.FromErr(err).Err()
	}

	log.Infoln("Creating Roles for ingress", lbc.Ingress.OffshootName())
	if err := lbc.ensureRoles(); err != nil {
		return errors.FromErr(err).Err()
	}

	log.Infoln("Creating RoleBinding for ingress", lbc.Ingress.OffshootName())
	if err := lbc.ensureRoleBinding(); err != nil {
		return errors.FromErr(err).Err()
	}

	return nil
}

func (lbc *Controller) createLB() error {
	if !lbc.SupportsLBType() {
		err := errors.Newf("LBType %s is unsupported for cloud provider: %s", lbc.Ingress.LBType(), lbc.Opt.CloudProvider).Err()
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeWarning,
			eventer.EventReasonIngressUnsupportedLBType,
			err.Error(),
		)
		return err
	}

	// Specifically Add Controller and Service Event Separately for all LBTypes.
	log.Infof("Creating Resource for ingress %s, LBType detected %s", lbc.Ingress.OffshootName(), lbc.Ingress.LBType())
	if lbc.Ingress.LBType() == api.LBTypeHostPort {
		err := lbc.createHostPortPods()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressControllerCreateFailed,
				"Failed to create HostPortPods, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully created HostPortPods",
		)

		time.Sleep(time.Second * 10)
		err = lbc.createHostPortSvc()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceCreateFailed,
				"Failed to create HostPortService, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully created HostPortService",
		)
	} else if lbc.Ingress.LBType() == api.LBTypeNodePort {
		err := lbc.createNodePortPods()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressControllerCreateFailed,
				"Failed to create NodePortPods, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully created NodePortPods")

		time.Sleep(time.Second * 10)
		err = lbc.createNodePortSvc()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceCreateFailed,
				"Failed to create NodePortService, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully created NodePortService",
		)
	} else {
		// deleteResidualPods is a safety checking deletion of previous version RC
		// This should Ignore error.
		lbc.deleteResidualPods()
		err := lbc.createNodePortPods()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressControllerCreateFailed,
				"Failed to create NodePortPods, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressControllerCreateSuccessful,
			"Successfully created NodePortPods",
		)

		time.Sleep(time.Second * 10)
		err = lbc.createLoadBalancerSvc()
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceCreateFailed,
				"Failed to create LoadBalancerService, Reason: %s",
				err.Error(),
			)
			return errors.FromErr(err).Err()
		}
		lbc.recorder.Eventf(
			lbc.Ingress,
			apiv1.EventTypeNormal,
			eventer.EventReasonIngressServiceCreateSuccessful,
			"Successfully created LoadBalancerService",
		)

		go lbc.updateStatus()
	}

	monSpec, err := lbc.Ingress.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(lbc.KubeClient, lbc.PromClient)
		err := ctrl.AddMonitor(lbc.Ingress, monSpec)
		// Error Ignored intentionally
		if err != nil {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeWarning,
				eventer.EventReasonIngressServiceMonitorCreateFailed,
				err.Error(),
			)
		} else {
			lbc.recorder.Eventf(
				lbc.Ingress,
				apiv1.EventTypeNormal,
				eventer.EventReasonIngressServiceMonitorCreateSuccessful,
				"Successfully created ServiceMonitor",
			)
		}
	}
	return nil
}

func (lbc *Controller) createHostPortSvc() error {
	// Create a Headless service without selectors
	// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},

		Spec: apiv1.ServiceSpec{
			Type:      apiv1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports:     []apiv1.ServicePort{},
		},
	}

	// opening other tcp ports
	for svcPort, targetPort := range lbc.Ports {
		p := apiv1.ServicePort{
			Name:       "tcp-" + strconv.Itoa(svcPort),
			Protocol:   "TCP",
			Port:       int32(svcPort),
			TargetPort: intstr.FromInt(targetPort),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Ingress.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	updateFW := false
	s, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		updateFW = true
	} else if err != nil {
		return errors.FromErr(err).Err()
	} else {
		needsUpdate := false
		if val, ok := lbc.ensureResourceAnnotations(s.Annotations); ok {
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
			_, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Update(s)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}

	if updateFW && lbc.CloudManager != nil {
		daemonNodes, err := lbc.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(lbc.Ingress.NodeSelector()).String(),
		})
		if err != nil {
			log.Infoln("node not found with nodeSelector, cause", err)
			return errors.FromErr(err).Err()
		}
		// open up firewall
		log.Debugln("Checking cloud manager", lbc.CloudManager)
		if fw, ok := lbc.CloudManager.Firewall(); ok {
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

func (lbc *Controller) createHostPortPods() error {
	log.Infoln("Creating Daemon type lb for nodeSelector = ", lbc.Ingress.NodeSelector())

	vs := Volumes(lbc.SecretNames)
	vms := VolumeMounts(lbc.SecretNames)
	// ignoring errors and trying to create controllers
	daemon := &extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Labels:    lbc.Ingress.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},

		Spec: extensions.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: lbc.Ingress.OffshootLabels(),
			},

			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbc.Ingress.OffshootLabels(),
				},
				Spec: apiv1.PodSpec{
					NodeSelector: lbc.Ingress.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: lbc.Opt.HAProxyImage,
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
								"--config-map=" + lbc.Ingress.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []apiv1.ContainerPort{},
							Resources:    lbc.Ingress.Spec.Resources,
							VolumeMounts: vms,
						},
					},
					Volumes:     vs,
					HostNetwork: true,
				},
			},
		},
	}

	if lbc.Opt.EnableRBAC {
		daemon.Spec.Template.Spec.ServiceAccountName = lbc.Ingress.OffshootName()
	}

	exporter, err := lbc.getExporterSidecar()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if exporter != nil {
		daemon.Spec.Template.Spec.Containers = append(daemon.Spec.Template.Spec.Containers, *exporter)
	}

	// adding tcp ports to pod template
	targetPorts := make(map[int]bool)
	for _, targetPort := range lbc.Ports {
		targetPorts[targetPort] = true
	}
	for targetPort := range targetPorts {
		p := apiv1.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(targetPort),
			Protocol:      "TCP",
			ContainerPort: int32(targetPort),
			HostPort:      int32(targetPort),
		}
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if lbc.Parsed.Stats {
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{
			Name:          api.StatsPortName,
			Protocol:      "TCP",
			ContainerPort: int32(lbc.Parsed.StatsPort),
		})
	}

	if ans, ok := lbc.Ingress.PodsAnnotations(); ok {
		daemon.Spec.Template.Annotations = ans
	}

	dm, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Get(daemon.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		log.Infoln("creating DaemonSets controller")
		_, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Create(daemon)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(dm.Annotations); ok {
		needsUpdate = true
		dm.Annotations = val
	}

	if !reflect.DeepEqual(dm.Spec, daemon.Spec) {
		needsUpdate = true
		dm.Spec = daemon.Spec
	}

	if needsUpdate {
		_, err = lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Update(dm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) createNodePortSvc() error {
	log.Infoln("creating NodePort type lb")
	// creating service as type NodePort
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:     apiv1.ServiceTypeNodePort,
			Ports:    []apiv1.ServicePort{},
			Selector: lbc.Ingress.OffshootLabels(),
			// https://github.com/kubernetes/kubernetes/issues/33586
			// LoadBalancerSourceRanges: lbc.Config.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	for svcPort, targetPort := range lbc.Ports {
		p := apiv1.ServicePort{
			Name:       "tcp-" + strconv.Itoa(svcPort),
			Protocol:   "TCP",
			Port:       int32(svcPort),
			TargetPort: intstr.FromInt(targetPort),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Ingress.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	if lbc.Opt.CloudProvider == "aws" && lbc.Ingress.KeepSourceIP() {
		// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
		svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"] = "*"
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(s.Annotations); ok {
		s.Annotations = val
		needsUpdate = true
	}

	if isServicePortChanged(s.Spec.Ports, svc.Spec.Ports) {
		needsUpdate = true
		s.Spec.Ports = svc.Spec.Ports
	}

	if needsUpdate {
		_, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Update(s)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) createNodePortPods() error {
	log.Infoln("creating NodePort deployment")
	deployment := &extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Labels:    lbc.Ingress.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},

		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(lbc.Ingress.Replicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: lbc.Ingress.OffshootLabels(),
			},
			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbc.Ingress.OffshootLabels(),
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
											MatchLabels: lbc.Ingress.OffshootLabels(),
										},
									},
								},
							},
						},
					},
					NodeSelector: lbc.Ingress.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: lbc.Opt.HAProxyImage,
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
								"--config-map=" + lbc.Ingress.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []apiv1.ContainerPort{},
							Resources:    lbc.Ingress.Spec.Resources,
							VolumeMounts: VolumeMounts(lbc.SecretNames),
						},
					},
					Volumes: Volumes(lbc.SecretNames),
				},
			},
		},
	}

	if lbc.Opt.EnableRBAC {
		deployment.Spec.Template.Spec.ServiceAccountName = lbc.Ingress.OffshootName()
	}

	exporter, err := lbc.getExporterSidecar()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if exporter != nil {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, *exporter)
	}

	// adding tcp ports to pod template
	targetPorts := make(map[int]bool)
	for _, targetPort := range lbc.Ports {
		targetPorts[targetPort] = true
	}
	for targetPort := range targetPorts {
		p := apiv1.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(targetPort),
			Protocol:      "TCP",
			ContainerPort: int32(targetPort),
		}
		deployment.Spec.Template.Spec.Containers[0].Ports = append(deployment.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if lbc.Parsed.Stats {
		deployment.Spec.Template.Spec.Containers[0].Ports = append(deployment.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{
			Name:          api.StatsPortName,
			Protocol:      "TCP",
			ContainerPort: int32(lbc.Parsed.StatsPort),
		})
	}

	if ans, ok := lbc.Ingress.PodsAnnotations(); ok {
		deployment.Spec.Template.Annotations = ans
	}

	dpl, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Get(deployment.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Create(deployment)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(dpl.Annotations); ok {
		needsUpdate = true
		dpl.Annotations = val
	}

	if !reflect.DeepEqual(dpl.Spec, deployment.Spec) {
		needsUpdate = true
		dpl.Spec = deployment.Spec
	}

	if needsUpdate {
		_, err = lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Update(dpl)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) createLoadBalancerSvc() error {
	log.Infoln("creating LoadBalancer type lb")
	// creating service as typeLoadBalancer
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:                     apiv1.ServiceTypeLoadBalancer,
			Ports:                    []apiv1.ServicePort{},
			Selector:                 lbc.Ingress.OffshootLabels(),
			LoadBalancerSourceRanges: lbc.Ingress.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	for svcPort, targetPort := range lbc.Ports {
		p := apiv1.ServicePort{
			Name:       "tcp-" + strconv.Itoa(svcPort),
			Protocol:   "TCP",
			Port:       int32(svcPort),
			TargetPort: intstr.FromInt(targetPort),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Ingress.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	switch lbc.Opt.CloudProvider {
	case "gce", "gke":
		if ip := lbc.Ingress.LoadBalancerIP(); ip != nil {
			svc.Spec.LoadBalancerIP = ip.String()
		}
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Create(svc)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		return nil
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(s.Annotations); ok {
		s.Annotations = val
		needsUpdate = true
	}

	if isServicePortChanged(s.Spec.Ports, svc.Spec.Ports) {
		needsUpdate = true
		s.Spec.Ports = svc.Spec.Ports
	}

	if needsUpdate {
		_, err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Update(s)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *Controller) getExporterSidecar() (*apiv1.Container, error) {
	if !lbc.Ingress.Stats() {
		return nil, nil // Don't add sidecar is stats is not exposed.
	}
	monSpec, err := lbc.Ingress.MonitorSpec()
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

func (lbc *Controller) ensureStatsService() error {
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.StatsServiceName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
			Labels: lbc.Ingress.StatsLabels(),
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name:       api.StatsPortName,
					Protocol:   "TCP",
					Port:       int32(lbc.Parsed.StatsPort),
					TargetPort: intstr.FromString(api.StatsPortName),
				},
			},
			Selector: lbc.Ingress.OffshootLabels(),
		},
	}
	monSpec, err := lbc.Ingress.MonitorSpec()
	if err == nil && monSpec != nil && monSpec.Prometheus != nil {
		svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
			Name:       api.ExporterPortName,
			Protocol:   "TCP",
			Port:       int32(monSpec.Prometheus.Port),
			TargetPort: intstr.FromString(api.ExporterPortName),
		})
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.StatsServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Create(svc)
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
	_, err = lbc.KubeClient.CoreV1().Services(s.Namespace).Update(s)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (lbc *Controller) updateStatus() error {
	var statuses []apiv1.LoadBalancerIngress

	switch lbc.Ingress.LBType() {
	case api.LBTypeLoadBalancer:
		for i := 0; i < 50; i++ {
			time.Sleep(time.Second * 10)
			if svc, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{}); err == nil {
				if len(svc.Status.LoadBalancer.Ingress) >= 1 {
					statuses = svc.Status.LoadBalancer.Ingress
					break
				}
			}
		}
		// TODO @sadlil consider adding node ip in status for hostport/nodeport mode
	}

	if len(statuses) > 0 {
		if lbc.Ingress.APISchema() == api.APISchemaIngress {
			ing, err := lbc.KubeClient.ExtensionsV1beta1().Ingresses(lbc.Ingress.Namespace).Get(lbc.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.KubeClient.ExtensionsV1beta1().Ingresses(lbc.Ingress.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			ing, err := lbc.ExtClient.Ingresses(lbc.Ingress.Namespace).Get(lbc.Ingress.Name)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.ExtClient.Ingresses(lbc.Ingress.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	return nil
}

func (lbc *Controller) ensureResourceAnnotations(annotation map[string]string) (map[string]string, bool) {
	needsUpdate := false

	// Copy the given map to avoid updating the original annotations
	ret := annotation
	if ret == nil {
		ret = make(map[string]string)
	}

	if val := ret[api.OriginAPISchema]; val != lbc.Ingress.APISchema() {
		needsUpdate = true
		ret[api.OriginAPISchema] = lbc.Ingress.APISchema()
	}

	if val := ret[api.OriginName]; val != lbc.Ingress.GetName() {
		needsUpdate = true
		ret[api.OriginName] = lbc.Ingress.GetName()
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
