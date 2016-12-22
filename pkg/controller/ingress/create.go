package ingress

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/encoding/yaml"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	extensions "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const NodeSelectorKey = "kubernetes.io/hostname"

func (lbc *EngressController) Create() error {
	log.Debugln("Starting createing lb. got engress with", lbc.Config.ObjectMeta)
	err := lbc.parse()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	if b, err := json.MarshalIndent(lbc.Options, "", "  "); err != nil {
		log.Infoln("Parsed LB controller options: ", string(b))
	}

	err = lbc.generateTemplate()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	// This methods clean up any unwanted resource that will cause in errors
	lbc.ensureResources()

	err = lbc.createConfigMap()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	err = lbc.createLB()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (lbc *EngressController) ensureResources() {
	// if there is already an resource with this name
	// delete those resource and create the new resource.
	log.Debugln("trying to delete already existing resources.")
	lbc.deleteConfigMap()
	lbc.deleteLB()
}

func (lbc *EngressController) createConfigMap() error {
	log.Infoln("creating cmap for engress")
	cMap := &kapi.ConfigMap{
		ObjectMeta: kapi.ObjectMeta{
			Name:      ConfigMapPrefix + lbc.Config.Name,
			Namespace: lbc.Config.Namespace,
		},
		Data: map[string]string{
			"haproxy.cfg": lbc.Options.ConfigData,
		},
	}
	cMap, err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Create(cMap)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	lbc.Options.ConfigMapName = cMap.Name
	time.Sleep(time.Second * 20)
	return nil
}

func (lbc *EngressController) createLB() error {
	var err error
	if lbc.Options.LBType == LBDaemon {
		err = lbc.createDaemonLB()
	} else {
		if lbc.Options.ProviderName == "aws" ||
			lbc.Options.ProviderName == "gce" ||
			lbc.Options.ProviderName == "azure" {
			err = lbc.createLoadBalancerLB()
		} else {
			err = errors.New().WithMessage("LoadBalancer type ingress is unsupported for cloud provider:", lbc.Options.ProviderName).BadRequest()
		}
	}
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (lbc *EngressController) createDaemonLB() error {
	log.Infoln("Creating Daemon type lb for host name = ", lbc.Options.DaemonHostname)
	daemonNode, err := lbc.KubeClient.Core().Nodes().Get(lbc.Options.DaemonHostname)
	if err != nil {
		log.Infoln("node not found with host name", err)
		if lbc.Options.ProviderName == "aws" {
			log.Infoln("provider is aws, trying to get suffix")
			hostName := ""
			hostName, err = fullyQualifiedNodeName(lbc.Options.DaemonHostname)
			if err != nil {
				log.Errorln("Error getting host name", err)
			} else {
				log.Infoln("host name found with", hostName)
				daemonNode, err = lbc.KubeClient.Core().Nodes().Get(hostName)
				if err != nil {
					log.Infoln("node still not found")
				}
			}
		}
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
	}
	// detect IP when CloudManager is found.
	if lbc.Options.LoadBalancerIP == "" {
		for _, addr := range daemonNode.Status.Addresses {
			if addr.Type == kapi.NodeInternalIP {
				lbc.Options.LoadBalancerIP = addr.Address
			}
		}
	}

	// Create a Headless service without selectors
	// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      ServicePrefix + lbc.Config.Name,
			Namespace: lbc.Config.Namespace,
			Annotations: map[string]string{
				LBName: lbc.Config.GetName(),
				LBType: LBDaemon,
			},
		},

		// default opening port 80 and 443 for the service.
		Spec: kapi.ServiceSpec{
			Type:      kapi.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports: []kapi.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
				{
					Name:       "https",
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
			// Selector: labelsFor(lbc.Config.Name),
			// ExternalIPs: []string{
			//	lbc.Options.LoadBalancerIP,
			// },
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

	svc, err = lbc.KubeClient.Core().Services(lbc.Config.Namespace).Create(svc)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	log.Infoln("creating daemonset")
	vs := Volumes(lbc.Options)
	vms := VolumeMounts(lbc.Options)
	// ignoring errors and trying to create controllers
	daemon := &extensions.DaemonSet{
		ObjectMeta: kapi.ObjectMeta{
			Name:      DaemonSetPrefix + lbc.Config.Name,
			Namespace: lbc.Config.Namespace,
			Labels:    labelsFor(lbc.Config.Name),
		},

		Spec: extensions.DaemonSetSpec{
			Selector: &unversioned.LabelSelector{
				MatchLabels: labelsFor(lbc.Config.Name),
			},

			// pod templates.
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: labelsFor(lbc.Config.Name),
				},
				Spec: kapi.PodSpec{
					NodeSelector: nodeSelector(lbc.Options.DaemonHostname),
					Containers: []kapi.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),

							Ports: []kapi.ContainerPort{
								{
									ContainerPort: 80,
									HostPort:      80,
									Protocol:      "TCP",
								},
								{
									ContainerPort: 443,
									HostPort:      443,
									Protocol:      "TCP",
								},
							},

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
	log.Infoln("creating deamonsets controller")
	_, err = lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Create(daemon)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	log.V(5).Infoln("DeamonSet Created with\n", yaml.ToString(daemon))

	// open up firewall
	log.Debugln("Checking cloud manager", lbc.CloudManager)
	if lbc.CloudManager != nil {
		log.Debugln("cloud manager not nil")
		if fw, ok := lbc.CloudManager.Firewall(); ok {
			log.Debugln("firewalls found")
			b2, err := json.MarshalIndent(daemonNode, "", " ")
			log.Infoln(">>>> ---- ", string(b2))
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			err = fw.EnsureFirewall(convertedSvc, daemonNode.Name) // lbc.Config.ObjectMeta.Annotations
			if err != nil {
				return errors.New().WithCause(err).Internal()
			}
			log.Debugln("getting firewalls for cloud manager failed")
		}
	}
	return nil
}

func (lbc *EngressController) createLoadBalancerLB() error {
	log.Infoln("creating LoadBalancer type lb")
	// creating service as typeLoadBalancer
	svc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      ServicePrefix + lbc.Config.Name,
			Namespace: lbc.Config.Namespace,
			Annotations: map[string]string{
				LBName: lbc.Config.GetName(),
				LBType: LBLoadBalancer,
			},
		},

		// default opening port 80 and 443 for the service.
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
				{
					Name:       "https",
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
			Selector: labelsFor(lbc.Config.Name),
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

	if lbc.Options.ProviderName == "gce" {
		svc.Spec.Type = kapi.ServiceTypeLoadBalancer
		svc.Spec.LoadBalancerIP = lbc.Options.LoadBalancerIP
	}
	if lbc.Options.ProviderName == "aws" {
		if lbc.Options.LoadBalancerPersist {
			// We are going manage the loadbalancer directly
			svc.Spec.Type = kapi.ServiceTypeNodePort
		} else {
			svc.Spec.Type = kapi.ServiceTypeLoadBalancer
		}
	}

	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Create(svc)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	log.Infoln("creating replication controller")
	vs := Volumes(lbc.Options)
	vms := VolumeMounts(lbc.Options)
	// ignoring errors and trying to create controllers
	rc := &kapi.ReplicationController{
		ObjectMeta: kapi.ObjectMeta{
			Name:      ControllerPrefix + lbc.Config.Name,
			Namespace: lbc.Config.Namespace,
			Labels:    labelsFor(lbc.Config.Name),
		},

		Spec: kapi.ReplicationControllerSpec{
			Replicas: 1,
			Selector: labelsFor(lbc.Config.Name),

			// pod templates.
			Template: &kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: labelsFor(lbc.Config.Name),
				},

				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:  "haproxy",
							Image: GetLoadbalancerImage(),
							Ports: []kapi.ContainerPort{
								{
									ContainerPort: 80,
									Protocol:      "TCP",
								},
								{
									ContainerPort: 443,
									Protocol:      "TCP",
								},
							},

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
		rc.Spec.Template.Spec.Containers[0].Ports = append(rc.Spec.Template.Spec.Containers[0].Ports, p)
	}
	log.Debugln("creating replication controller")
	_, err = lbc.KubeClient.Core().ReplicationControllers(lbc.Config.Namespace).Create(rc)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	if svc.Spec.Type == kapi.ServiceTypeNodePort && lbc.CloudManager != nil {
		log.Debugln("cloud manager not il, getting hosts")
		hosts := make([]string, 0)
		if ins, ok := lbc.CloudManager.Instances(); ok {
			// TODO(tamal): Does it return all hosts?
			nodes, _ := ins.List("")
			for _, node := range nodes {
				hosts = append(hosts, string(node))
			}
		}

		log.Debugln("hosts found", hosts)
		// Wait for nodePort to be assigned
		timeoutAt := time.Now().Add(time.Second * 600)
		for {
			svc, _ := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(ServicePrefix + lbc.Config.Name)
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
				return errors.New().WithMessage("timed out creating node port service").Failed()
			}

			log.Info("Waiting for nodeport service to be ready")

			time.Sleep(10 * time.Second)
		}
		if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
			log.Debugln("loadbalancer for cloud manager updating")
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			_, err = lb.EnsureLoadBalancer(lbc.Options.ClusterName, convertedSvc, hosts) // lbc.Config.Annotations
			if err != nil {
				return errors.New().WithCause(err).Internal()
			}
		}
	}
	return nil
}

func (lbc *EngressController) updateEngressIP() error {
	var serviceExtIp string
	for {
		time.Sleep(time.Second * 60)
		if svc, err := lbc.KubeClient.Core().
			Services(lbc.Config.Namespace).
			Get(ServicePrefix + lbc.Config.Name); err == nil {
			if len(svc.Status.LoadBalancer.Ingress) >= 1 {
				serviceExtIp = svc.Status.LoadBalancer.Ingress[0].IP
				break
			}
		}
	}
	if len(lbc.Config.Status.LoadBalancer.Ingress) <= 0 {
		lbc.Config.Status.LoadBalancer.Ingress = make([]kapi.LoadBalancerIngress, 0)
	}
	log.Infoln("Updating engress")
	lbc.Config.Status.LoadBalancer.Ingress = append(lbc.Config.Status.LoadBalancer.Ingress, kapi.LoadBalancerIngress{IP: serviceExtIp})
	_, err := lbc.ACExtensionClient.Ingress(lbc.Config.Namespace).Update(lbc.Config)
	if err != nil {
		return errors.New().WithCause(err).Internal()
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

func nodeSelector(name string) map[string]string {
	return map[string]string{
		NodeSelectorKey: name,
	}
}

func Volumes(o *KubeOptions) []kapi.Volume {
	configVolume := kapi.Volume{
		Name: "haproxy-configs",
		VolumeSource: kapi.VolumeSource{
			ConfigMap: &kapi.ConfigMapVolumeSource{
				LocalObjectReference: kapi.LocalObjectReference{
					Name: o.ConfigMapName,
				},
			},
		},
	}

	skipper := make(map[string]bool)
	vs := make([]kapi.Volume, 0)
	vs = append(vs, configVolume)
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
	configMount := kapi.VolumeMount{
		Name:      "haproxy-configs",
		MountPath: "/etc/haproxy",
	}
	skipper := make(map[string]bool)
	ms := make([]kapi.VolumeMount, 0)
	ms = append(ms, configMount)
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
