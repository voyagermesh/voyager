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
	"github.com/appscode/voyager/pkg/monitor"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	ExporterSidecarTag = "appscode/voyager:3.0.0"
)

func (lbc *IngressController) Create() error {
	log.Debugln("Starting creating lb. got engress with", lbc.Resource.ObjectMeta)
	err := lbc.parse()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	err = lbc.ensureConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	time.Sleep(time.Second * 5)
	err = lbc.createLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if lbc.Parsed.Stats {
		lbc.ensureStatsService()
	}
	return nil
}

func (lbc *IngressController) ensureConfigMap() error {
	log.Infoln("creating cmap for engress")
	cm, err := lbc.KubeClient.CoreV1().ConfigMaps(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		cm = &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      lbc.Resource.OffshootName(),
				Namespace: lbc.Resource.Namespace,
				Annotations: map[string]string{
					api.OriginAPISchema: lbc.Resource.APISchema(),
					api.OriginName:      lbc.Resource.GetName(),
				},
			},
			Data: map[string]string{
				"haproxy.cfg": lbc.ConfigData,
			},
		}
		_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Resource.Namespace).Create(cm)
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
		_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Resource.Namespace).Update(cm)
		return err
	}
	return nil
}

func (lbc *IngressController) createLB() error {
	if !lbc.SupportsLBType() {
		return errors.Newf("LBType %s is unsupported for cloud provider: %s", lbc.Resource.LBType(), lbc.Opt.CloudProvider).Err()
	}

	if lbc.Resource.LBType() == api.LBTypeHostPort {
		err := lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 10)
		err = lbc.createHostPortSvc()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Resource.LBType() == api.LBTypeNodePort {
		err := lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 10)
		err = lbc.createNodePortSvc()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		// deleteResidualPods is a safety checking deletion of previous version RC
		// This should Ignore error.
		lbc.deleteResidualPods()
		err := lbc.createNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 10)
		err = lbc.createLoadBalancerSvc()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		go lbc.updateStatus()
	}

	monSpec, err := lbc.Resource.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(lbc.KubeClient, lbc.PromClient)
		ctrl.AddMonitor(lbc.Resource, monSpec)
	}
	return nil
}

func (lbc *IngressController) createHostPortSvc() error {
	// Create a Headless service without selectors
	// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
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

	if ans, ok := lbc.Resource.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	updateFW := false
	s, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Create(svc)
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
			_, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Update(s)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}

	if updateFW && lbc.CloudManager != nil {
		daemonNodes, err := lbc.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(lbc.Resource.NodeSelector()).String(),
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

func (lbc *IngressController) createHostPortPods() error {
	if len(lbc.Resource.NodeSelector()) == 0 {
		return errors.Newf("%s type ingress %s@%s is missing node selectors.", lbc.Resource.LBType(), lbc.Resource.Name, lbc.Resource.Namespace).Err()
	}
	log.Infoln("Creating Daemon type lb for nodeSelector = ", lbc.Resource.NodeSelector())

	vs := Volumes(lbc.SecretNames)
	vms := VolumeMounts(lbc.SecretNames)
	// ignoring errors and trying to create controllers
	daemon := &extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Labels:    lbc.Resource.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
			},
		},

		Spec: extensions.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: lbc.Resource.OffshootLabels(),
			},

			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbc.Resource.OffshootLabels(),
				},
				Spec: apiv1.PodSpec{
					NodeSelector: lbc.Resource.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),
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
								"--config-map=" + lbc.Resource.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []apiv1.ContainerPort{},
							VolumeMounts: vms,
						},
					},
					Volumes:     vs,
					HostNetwork: true,
				},
			},
		},
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

	if ans, ok := lbc.Resource.PodsAnnotations(); ok {
		daemon.Spec.Template.Annotations = ans
	}

	dm, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Resource.Namespace).Get(daemon.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		log.Infoln("creating DaemonSets controller")
		_, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Resource.Namespace).Create(daemon)
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
		_, err = lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Resource.Namespace).Update(dm)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *IngressController) createNodePortSvc() error {
	log.Infoln("creating NodePort type lb")
	// creating service as type NodePort
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:     apiv1.ServiceTypeNodePort,
			Ports:    []apiv1.ServicePort{},
			Selector: lbc.Resource.OffshootLabels(),
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

	if ans, ok := lbc.Resource.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	if lbc.Opt.CloudProvider == "aws" && lbc.Resource.KeepSourceIP() {
		// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
		svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"] = "*"
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Create(svc)
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
		_, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Update(s)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *IngressController) createNodePortPods() error {
	log.Infoln("creating NodePort deployment")
	deployment := &extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Labels:    lbc.Resource.OffshootLabels(),
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
			},
		},

		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(lbc.Resource.Replicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: lbc.Resource.OffshootLabels(),
			},
			// pod templates.
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbc.Resource.OffshootLabels(),
				},
				Spec: apiv1.PodSpec{
					Affinity: &apiv1.Affinity{
						PodAntiAffinity: &apiv1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: apiv1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: lbc.Resource.OffshootLabels(),
										},
									},
								},
							},
						},
					},
					NodeSelector: lbc.Resource.NodeSelector(),
					Containers: []apiv1.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),
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
								"--config-map=" + lbc.Resource.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []apiv1.ContainerPort{},
							VolumeMounts: VolumeMounts(lbc.SecretNames),
						},
					},
					Volumes: Volumes(lbc.SecretNames),
				},
			},
		},
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

	if ans, ok := lbc.Resource.PodsAnnotations(); ok {
		deployment.Spec.Template.Annotations = ans
	}

	dpl, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Resource.Namespace).Get(deployment.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Resource.Namespace).Create(deployment)
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
		_, err = lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Resource.Namespace).Update(dpl)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *IngressController) createLoadBalancerSvc() error {
	log.Infoln("creating LoadBalancer type lb")
	// creating service as typeLoadBalancer
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:                     apiv1.ServiceTypeLoadBalancer,
			Ports:                    []apiv1.ServicePort{},
			Selector:                 lbc.Resource.OffshootLabels(),
			LoadBalancerSourceRanges: lbc.Resource.Spec.LoadBalancerSourceRanges,
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

	if ans, ok := lbc.Resource.ServiceAnnotations(lbc.Opt.CloudProvider); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	switch lbc.Opt.CloudProvider {
	case "gce", "gke":
		if ip := lbc.Resource.LoadBalancerIP(); ip != nil {
			svc.Spec.LoadBalancerIP = ip.String()
		}
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(svc.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		svc, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Create(svc)
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
		_, err = lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Update(s)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}
	return nil
}

func (lbc *IngressController) getExporterSidecar() (*apiv1.Container, error) {
	if !lbc.Resource.Stats() {
		return nil, nil // Don't add sidecar is stats is not exposed.
	}
	monSpec, err := lbc.Resource.MonitorSpec()
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

func (lbc *IngressController) ensureStatsService() {
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Resource.StatsServiceName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Resource.APISchema(),
				api.OriginName:      lbc.Resource.GetName(),
			},
			Labels: lbc.Resource.StatsLabels(),
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
			Selector: lbc.Resource.OffshootLabels(),
		},
	}
	monSpec, err := lbc.Resource.MonitorSpec()
	if err == nil && monSpec != nil && monSpec.Prometheus != nil {
		svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
			Name:       api.ExporterPortName,
			Protocol:   "TCP",
			Port:       int32(monSpec.Prometheus.Port),
			TargetPort: intstr.FromString(api.ExporterPortName),
		})
	}

	s, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(lbc.Resource.StatsServiceName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Create(svc)
		if err != nil {
			log.Errorln("Failed to create Stats Service", err)
		}
		return
	} else if err != nil {
		log.Errorln(err)
		return
	}
	s.Labels = svc.Labels
	s.Annotations = svc.Annotations
	s.Spec = svc.Spec
	_, err = lbc.KubeClient.CoreV1().Services(s.Namespace).Update(s)
	if err != nil {
		log.Errorln("Failed to update Stats Service", err)
	}
}

func (lbc *IngressController) updateStatus() error {
	var statuses []apiv1.LoadBalancerIngress

	switch lbc.Resource.LBType() {
	case api.LBTypeLoadBalancer:
		for i := 0; i < 50; i++ {
			time.Sleep(time.Second * 10)
			if svc, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{}); err == nil {
				if len(svc.Status.LoadBalancer.Ingress) >= 1 {
					statuses = svc.Status.LoadBalancer.Ingress
					break
				}
			}
		}
		// TODO @sadlil consider adding node ip in status for hostport/nodeport mode
	}

	if len(statuses) > 0 {
		if lbc.Resource.APISchema() == api.APISchemaIngress {
			ing, err := lbc.KubeClient.ExtensionsV1beta1().Ingresses(lbc.Resource.Namespace).Get(lbc.Resource.Name, metav1.GetOptions{})
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.KubeClient.ExtensionsV1beta1().Ingresses(lbc.Resource.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			ing, err := lbc.ExtClient.Ingresses(lbc.Resource.Namespace).Get(lbc.Resource.Name)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.ExtClient.Ingresses(lbc.Resource.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	return nil
}

func (lbc *IngressController) ensureResourceAnnotations(annotation map[string]string) (map[string]string, bool) {
	needsUpdate := false

	// Copy the given map to avoid updating the original annotations
	ret := annotation
	if ret == nil {
		ret = make(map[string]string)
	}

	if val := ret[api.OriginAPISchema]; val != lbc.Resource.APISchema() {
		needsUpdate = true
		ret[api.OriginAPISchema] = lbc.Resource.APISchema()
	}

	if val := ret[api.OriginName]; val != lbc.Resource.GetName() {
		needsUpdate = true
		ret[api.OriginName] = lbc.Resource.GetName()
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
