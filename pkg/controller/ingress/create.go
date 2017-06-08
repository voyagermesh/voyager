package ingress

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/encoding/yaml"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kepi "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func (lbc *EngressController) Create() error {
	log.Debugln("Starting creating lb. got engress with", lbc.Resource.ObjectMeta)
	err := lbc.parse()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if b, err := json.MarshalIndent(lbc.Options, "", "  "); err != nil {
		log.Infoln("Parsed LB controller options: ", string(b))
	}

	err = lbc.generateTemplate()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	// This methods clean up any unwanted resource that will cause in errors
	lbc.ensureResources()

	err = lbc.createConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = lbc.createLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if lbc.Parsed.Stats {
		lbc.ensureStatsService()
	}
	return nil
}

func (lbc *EngressController) ensureResources() {
	// if there is already an resource with this name
	// delete those resource and create the new resource.
	log.Debugln("trying to delete already existing resources.")
	_, err := lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Get(lbc.OffshootName())
	if err == nil {
		lbc.deleteConfigMap()
	}
	if lbc.Annotations().LBType() == LBTypeDaemon || lbc.Annotations().LBType() == LBTypeHostPort {
		_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Get(lbc.OffshootName())
		if err == nil {
			lbc.deleteHostPortPods()
		}
	} else if lbc.Annotations().LBType() == LBTypeNodePort {
		_, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(lbc.OffshootName())
		if err == nil {
			lbc.deleteNodePortPods()

		}
	} else {
		// Ignore Error.
		lbc.deleteResidualPods()
		_, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(lbc.OffshootName())
		if err == nil {
			lbc.deleteNodePortPods()
		}
	}
	lbc.deleteLBSvc()
}

func (lbc *EngressController) createConfigMap() error {
	log.Infoln("creating cmap for engress")
	cMap := &kapi.ConfigMap{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},
		Data: map[string]string{
			"haproxy.cfg": lbc.Options.ConfigData,
		},
	}
	cMap, err := lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Create(cMap)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	time.Sleep(time.Second * 20)
	return nil
}

func (lbc *EngressController) createLB() error {
	if lbc.Annotations().LBType() == LBTypeDaemon || lbc.Annotations().LBType() == LBTypeHostPort {
		err := lbc.createHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 10)
		err = lbc.createHostPortSvc()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Annotations().LBType() == LBTypeNodePort {
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
		if lbc.Options.SupportsLoadBalancerType() {
			// deleteResidualPods is a safety checking deletation of previous version RC
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
		} else {
			return errors.New("LoadBalancer type ingress is unsupported for cloud provider:", lbc.Options.ProviderName).Err()
		}
	}
	return nil
}

func (lbc *EngressController) createHostPortSvc() error {
	// Create a Headless service without selectors
	// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},

		Spec: kapi.ServiceSpec{
			Type:      kapi.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports:     []kapi.ServicePort{},
		},
	}

	// opening other tcp ports
	for _, port := range lbc.Options.Ports {
		p := kapi.ServicePort{
			Name:       "tcp-" + strconv.Itoa(port),
			Protocol:   "TCP",
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Annotations().ServiceAnnotations(lbc.Options.ProviderName, lbc.Annotations().LBType()); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Create(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	daemonNodes, err := lbc.KubeClient.Core().Nodes().List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(lbc.Annotations().NodeSelector())),
	})
	if err != nil {
		log.Infoln("node not found with nodeSelector, cause", err)
		return errors.FromErr(err).Err()
	}
	// open up firewall
	log.Debugln("Checking cloud manager", lbc.CloudManager)
	if lbc.CloudManager != nil {
		log.Debugln("cloud manager not nil")
		if fw, ok := lbc.CloudManager.Firewall(); ok {
			log.Debugln("firewalls found")
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			for _, node := range daemonNodes.Items {
				err = fw.EnsureFirewall(convertedSvc, node.Name)
				if err != nil {
					log.Errorln("Failed to ensure loadbalancer for node", node.Name, "cause", err)
				}
			}
			log.Debugln("getting firewalls for cloud manager failed")
		}
	}
	return nil
}

func (lbc *EngressController) createHostPortPods() error {
	if len(lbc.Annotations().NodeSelector()) == 0 {
		return errors.Newf("%s type ingress %s@%s is missing node selectors.", lbc.Annotations().LBType(), lbc.Resource.Name, lbc.Resource.Namespace).Err()
	}
	log.Infoln("Creating Daemon type lb for nodeSelector = ", lbc.Annotations().NodeSelector())

	vs := Volumes(lbc.Options)
	vms := VolumeMounts(lbc.Options)
	// ignoring errors and trying to create controllers
	daemon := &kepi.DaemonSet{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Labels:    labelsFor(lbc.Resource.Name),
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},

		Spec: kepi.DaemonSetSpec{
			Selector: &unversioned.LabelSelector{
				MatchLabels: labelsFor(lbc.Resource.Name),
			},

			// pod templates.
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: labelsFor(lbc.Resource.Name),
				},
				Spec: kapi.PodSpec{
					NodeSelector: lbc.Annotations().NodeSelector(),
					Containers: []kapi.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),
							Env: []kapi.EnvVar{
								{
									Name: "KUBE_NAMESPACE",
									ValueFrom: &kapi.EnvVarSource{
										FieldRef: &kapi.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Args: []string{
								"--config-map=" + lbc.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []kapi.ContainerPort{},
							VolumeMounts: vms,
						},
					},
					Volumes: vs,
					SecurityContext: &kapi.PodSecurityContext{
						HostNetwork: true,
					},
				},
			},
		},
	}

	// adding tcp ports to pod template
	for _, port := range lbc.Options.Ports {
		p := kapi.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(port),
			Protocol:      "TCP",
			ContainerPort: int32(port),
			HostPort:      int32(port),
		}
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if lbc.Parsed.Stats {
		daemon.Spec.Template.Spec.Containers[0].Ports = append(daemon.Spec.Template.Spec.Containers[0].Ports, kapi.ContainerPort{
			Name:          "stats",
			Protocol:      "TCP",
			ContainerPort: int32(lbc.Parsed.StatsPort),
			HostPort:      int32(lbc.Parsed.StatsPort),
		})
	}

	if ans, ok := lbc.Annotations().PodsAnnotations(); ok {
		daemon.Spec.Template.Annotations = ans
	}

	log.Infoln("creating DaemonSets controller")
	_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Create(daemon)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.V(5).Infoln("DaemonSet Created with\n", yaml.ToString(daemon))
	return nil
}

func (lbc *EngressController) createNodePortSvc() error {
	log.Infoln("creating NodePort type lb")
	// creating service as type NodePort
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},
		Spec: kapi.ServiceSpec{
			Type:     kapi.ServiceTypeNodePort,
			Ports:    []kapi.ServicePort{},
			Selector: labelsFor(lbc.Resource.Name),
			// https://github.com/kubernetes/kubernetes/issues/33586
			// LoadBalancerSourceRanges: lbc.Config.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	for _, port := range lbc.Options.Ports {
		p := kapi.ServicePort{
			Name:       "tcp-" + strconv.Itoa(port),
			Protocol:   "TCP",
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Annotations().ServiceAnnotations(lbc.Options.ProviderName, lbc.Annotations().LBType()); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	if lbc.Options.ProviderName == "aws" && lbc.Annotations().KeepSourceIP() {
		// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
		svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"] = "*"
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Create(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (lbc *EngressController) createNodePortPods() error {
	log.Infoln("creating NodePort deployment")
	vs := Volumes(lbc.Options)
	vms := VolumeMounts(lbc.Options)
	// ignoring errors and trying to create controllers
	d := &kepi.Deployment{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Labels:    labelsFor(lbc.Resource.Name),
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},

		Spec: kepi.DeploymentSpec{
			Replicas: lbc.Annotations().Replicas(),
			Selector: &unversioned.LabelSelector{
				MatchLabels: labelsFor(lbc.Resource.Name),
			},
			// pod templates.
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: labelsFor(lbc.Resource.Name),
				},

				Spec: kapi.PodSpec{
					NodeSelector: lbc.Annotations().NodeSelector(),
					Containers: []kapi.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),
							Env: []kapi.EnvVar{
								{
									Name: "KUBE_NAMESPACE",
									ValueFrom: &kapi.EnvVarSource{
										FieldRef: &kapi.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Args: []string{
								"--config-map=" + lbc.OffshootName(),
								"--mount-location=" + "/etc/haproxy",
								"--boot-cmd=" + "/etc/sv/reloader/reload",
								"--v=4",
							},
							Ports:        []kapi.ContainerPort{},
							VolumeMounts: vms,
						},
					},
					Volumes: vs,
				},
			},
		},
	}

	// adding tcp ports to pod template
	for _, port := range lbc.Options.Ports {
		p := kapi.ContainerPort{
			Name:          "tcp-" + strconv.Itoa(port),
			Protocol:      "TCP",
			ContainerPort: int32(port),
		}
		d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports, p)
	}

	if lbc.Parsed.Stats {
		d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports, kapi.ContainerPort{
			Name:          "stats",
			Protocol:      "TCP",
			ContainerPort: int32(lbc.Parsed.StatsPort),
			HostPort:      int32(lbc.Parsed.StatsPort),
		})
	}

	if ans, ok := lbc.Annotations().PodsAnnotations(); ok {
		d.Spec.Template.Annotations = ans
	}

	_, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Create(d)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (lbc *EngressController) createLoadBalancerSvc() error {
	log.Infoln("creating LoadBalancer type lb")
	// creating service as typeLoadBalancer
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.OffshootName(),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},
		Spec: kapi.ServiceSpec{
			Ports:                    []kapi.ServicePort{},
			Selector:                 labelsFor(lbc.Resource.Name),
			LoadBalancerSourceRanges: lbc.Resource.Spec.LoadBalancerSourceRanges,
		},
	}

	// opening other tcp ports
	for _, port := range lbc.Options.Ports {
		p := kapi.ServicePort{
			Name:       "tcp-" + strconv.Itoa(port),
			Protocol:   "TCP",
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
		svc.Spec.Ports = append(svc.Spec.Ports, p)
	}

	if ans, ok := lbc.Annotations().ServiceAnnotations(lbc.Options.ProviderName, lbc.Annotations().LBType()); ok {
		for k, v := range ans {
			svc.Annotations[k] = v
		}
	}

	switch lbc.Options.ProviderName {
	case "gce", "gke":
		svc.Spec.Type = kapi.ServiceTypeLoadBalancer
		if ip := net.ParseIP(lbc.Annotations().LoadBalancerPersist()); ip != nil {
			svc.Spec.LoadBalancerIP = ip.String()
		}
	case "aws":
		if lbc.Annotations().LoadBalancerPersist() != "" {
			// We are going manage the loadbalancer directly
			svc.Spec.Type = kapi.ServiceTypeNodePort
		} else {
			svc.Spec.Type = kapi.ServiceTypeLoadBalancer
		}
	case "minikube":
		svc.Spec.Type = kapi.ServiceTypeLoadBalancer
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Create(svc)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if svc.Spec.Type == kapi.ServiceTypeNodePort && lbc.CloudManager != nil {
		if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
			// Wait for nodePort to be assigned
			timeoutAt := time.Now().Add(time.Second * 600)
			for {
				svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.OffshootName())
				if err != nil {
					return errors.FromErr(err).Err()
				}

				nodePortReady := true
				for _, p := range svc.Spec.Ports {
					if p.NodePort <= 0 {
						nodePortReady = false
						break
					}
				}
				if nodePortReady {
					break
				}

				if time.Now().After(timeoutAt) {
					return errors.New("timed out creating node port service").Err()
				}

				log.Info("Waiting for nodeport service to be ready")

				time.Sleep(10 * time.Second)
			}

			hosts := make([]string, 0)
			if ins, ok := lbc.CloudManager.Instances(); ok {
				// TODO(tamal): Does it return all hosts?
				nodes, _ := ins.List("")
				for _, node := range nodes {
					hosts = append(hosts, string(node))
				}
			}
			log.Debugln("loadbalancer for cloud manager updating")
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			_, err = lb.EnsureLoadBalancer(lbc.Options.ClusterName, convertedSvc, hosts) // lbc.Config.Annotations
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	return nil
}

func (lbc *EngressController) ensureStatsService() {
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      lbc.Annotations().StatsServiceName(lbc.Resource.GetName()),
			Namespace: lbc.Resource.Namespace,
			Annotations: map[string]string{
				OriginAPISchema: lbc.APISchema(),
				OriginName:      lbc.Resource.GetName(),
			},
		},
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{

					Name:       "stats",
					Protocol:   "TCP",
					Port:       int32(lbc.Parsed.StatsPort),
					TargetPort: intstr.FromInt(lbc.Parsed.StatsPort),
				},
			},
			Selector: labelsFor(lbc.Resource.Name),
		},
	}

	_, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Create(svc)
	if err != nil {
		log.Errorln("Failed to create Stats Service", err)
	}
}

func (lbc *EngressController) updateStatus() error {
	if lbc.Annotations().LBType() != LBTypeLoadBalancer {
		return nil
	}

	var statuses []kapi.LoadBalancerIngress
	for i := 0; i < 50; i++ {
		time.Sleep(time.Second * 10)
		if svc, err := lbc.KubeClient.Core().
			Services(lbc.Resource.Namespace).
			Get(lbc.OffshootName()); err == nil {
			if len(svc.Status.LoadBalancer.Ingress) >= 1 {
				statuses = svc.Status.LoadBalancer.Ingress
				break
			}
		}
	}

	if len(statuses) > 0 {
		if lbc.APISchema() == api.APISchemaIngress {
			ing, err := lbc.KubeClient.Extensions().Ingresses(lbc.Resource.Namespace).Get(lbc.Resource.Name)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.KubeClient.Extensions().Ingresses(lbc.Resource.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		} else {
			ing, err := lbc.ExtClient.Ingress(lbc.Resource.Namespace).Get(lbc.Resource.Name)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = lbc.ExtClient.Ingress(lbc.Resource.Namespace).Update(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	return nil
}

func labelsFor(name string) map[string]string {
	return map[string]string{
		"appType":     "ext-applbc-" + name,
		"type":        "ext-lbc-" + name,
		"target":      "eng-" + name,
		"meta":        "eng-" + name + "-applbc",
		"engressName": name,
	}
}

func Volumes(o *KubeOptions) []kapi.Volume {
	skipper := make(map[string]bool)
	vs := make([]kapi.Volume, 0)
	for _, s := range o.SecretNames {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, ok := skipper[s+"-secret-volume"]; ok {
			continue
		}
		skipper[s+"-secret-volume"] = true
		sVolume := kapi.Volume{
			Name: s + "-secret-volume",
			VolumeSource: kapi.VolumeSource{
				Secret: &kapi.SecretVolumeSource{
					SecretName: s,
				},
			},
		}
		vs = append(vs, sVolume)
	}
	return vs
}

func VolumeMounts(o *KubeOptions) []kapi.VolumeMount {
	skipper := make(map[string]bool)
	ms := make([]kapi.VolumeMount, 0)
	for _, s := range o.SecretNames {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, ok := skipper[s+"-secret-volume"]; ok {
			continue
		}
		skipper[s+"-secret-volume"] = true
		sMount := kapi.VolumeMount{
			Name:      s + "-secret-volume",
			MountPath: "/srv/haproxy/secrets/" + s,
		}
		ms = append(ms, sMount)
	}
	return ms
}
