package ingress

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	wpi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/analytics"
	"github.com/appscode/kutil/tools/cli"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/appscode/voyager/client/clientset/versioned/typed/voyager/v1beta1/util"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"kmodules.xyz/monitoring-agent-api/parser"
)

type loadBalancerController struct {
	*controller
}

var _ Controller = &loadBalancerController{}

func NewLoadBalancerController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	workloadClient wcs.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	cfg config.Config,
	ingress *api.Ingress,
	recorder record.EventRecorder) Controller {
	return &loadBalancerController{
		controller: &controller{
			logger:          log.New(ctx),
			KubeClient:      kubeClient,
			WorkloadClient:  workloadClient,
			CRDClient:       crdClient,
			VoyagerClient:   extClient,
			PromClient:      promClient,
			ServiceLister:   serviceLister,
			EndpointsLister: endpointsLister,
			cfg:             cfg,
			Ingress:         ingress,
			recorder:        recorder,
		},
	}
}

func (c *loadBalancerController) Reconcile() error {
	if err := c.generateConfig(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressHAProxyConfigReconcileFailed,
			"Reason: %v",
			err,
		)
		return errors.WithStack(err)
	}

	if _, vt, err := c.ensureConfigMap(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressConfigMapReconcileFailed,
			"Failed to reconcile ConfigMap %s, Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return errors.WithStack(err)
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressConfigMapReconcileSuccessful,
			"Successfully %s ConfigMap %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	// If RBAC is enabled we need to ensure service account
	if c.cfg.EnableRBAC {
		c.reconcileRBAC()
	}

	if vt, err := c.ensurePods(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressDeploymentReconcileFailed,
			"Failed to reconcile HAProxy Deployment %s, Reason: %s",
			c.Ingress.OffshootName(),
			err,
		)
		return errors.WithStack(err)
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressDeploymentReconcileSuccessful,
			"Successfully %s HAProxy Deployment %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	if _, vt, err := c.ensureService(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressServiceReconcileFailed,
			"Failed to reconcile LoadBalancer Service %s, Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return errors.WithStack(err)
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressServiceReconcileSuccessful,
			"Successfully %s LoadBalancer Service %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	go c.updateStatus()

	if c.Ingress.Stats() {
		if _, vt, err := c.ensureStatsService(); err != nil { // Error ignored intentionally
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressStatsServiceReconcileFailed,
				"Failed to ensure stats Service %s. Reason: %v",
				c.Ingress.StatsServiceName(),
				err,
			)
		} else if vt != kutil.VerbUnchanged {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceReconcileSuccessful,
				"Successfully %s stats Service %s",
				vt,
				c.Ingress.StatsServiceName(),
			)
		}
	} else {
		if err := c.ensureStatsServiceDeleted(); err != nil { // Error ignored intentionally
			log.Warningf("failed to delete stats Service %s, reason: %s", c.Ingress.StatsServiceName(), err)
		} else {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressStatsServiceDeleteSuccessful,
				"Successfully deleted stats Service %s",
				c.Ingress.StatsServiceName(),
			)
		}
	}

	monSpec, err := parser.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
	if err != nil {
		return errors.WithStack(err)
	}
	if monSpec != nil && c.Ingress.Stats() {
		if vt, err := c.ensureMonitoringAgent(monSpec); err != nil {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonIngressMonitorAgentReconcileFailed,
				"Failed to reconcile monitoring agent. Reason: %v",
				err,
			)
		} else if vt != kutil.VerbUnchanged {
			c.recorder.Eventf(
				c.Ingress.ObjectReference(),
				core.EventTypeNormal,
				eventer.EventReasonIngressMonitorAgentReconcileSuccessful,
				"Successfully %s monitoring agent",
				vt,
			)
		}
	} else { // monitoring disabled, delete old agent, ignore error here
		if err := c.ensureMonitoringAgentDeleted(nil); err != nil {
			log.Warningf("failed to delete old monitoring agent, reason: %s", err)
		}
	}

	return nil
}

func (c *loadBalancerController) EnsureFirewall(svc *core.Service) error {
	return nil
}

// make sure all delete calls require only ingress name and namespace
func (c *loadBalancerController) Delete() {
	if err := c.deletePods(); err != nil {
		c.logger.Errorln(err)
	}
	if err := c.deleteConfigMap(); err != nil {
		c.logger.Errorln(err)
	}
	if c.cfg.EnableRBAC {
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

func (c *loadBalancerController) ensureService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchService(c.KubeClient, meta, func(obj *core.Service) *core.Service {
		obj.ObjectMeta = c.ensureOwnerReference(obj.ObjectMeta)
		obj.Spec.Type = core.ServiceTypeLoadBalancer
		obj.Spec.Selector = c.Ingress.OffshootSelector()

		obj.Labels = c.Ingress.OffshootLabels()
		// Annotations
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		// delete last applied ServiceAnnotations
		// add new ServiceAnnotations
		// store new ServiceAnnotations keys
		lastAppliedKeys, _ := meta_util.GetStringValue(obj.Annotations, api.LastAppliedAnnotationKeys)
		for _, key := range strings.Split(lastAppliedKeys, ",") {
			delete(obj.Annotations, key)
		}
		newKeys := make([]string, 0)
		if ans, ok := c.Ingress.ServiceAnnotations(c.cfg.CloudProvider); ok {
			for k, v := range ans {
				obj.Annotations[k] = v
				newKeys = append(newKeys, k)
			}
		}
		sort.Strings(newKeys)
		obj.Annotations[api.LastAppliedAnnotationKeys] = strings.Join(newKeys, ",")

		// Remove old annotations from 3.2.x release.
		// ref: https://github.com/appscode/voyager/issues/527
		// https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/
		delete(obj.Annotations, "service.beta.kubernetes.io/external-traffic")
		delete(obj.Annotations, "service.beta.kubernetes.io/healthcheck-nodeport")

		// LoadBalancer ranges
		curRanges := sets.NewString()
		for _, ips := range obj.Spec.LoadBalancerSourceRanges {
			if k, ok := ipnet(ips); ok {
				curRanges.Insert(k)
			}
		}
		desiredRanges := sets.NewString()
		for _, ips := range c.Ingress.Spec.LoadBalancerSourceRanges {
			if k, ok := ipnet(ips); ok {
				desiredRanges.Insert(k)
			}
		}
		if !curRanges.Equal(desiredRanges) {
			obj.Spec.LoadBalancerSourceRanges = c.Ingress.Spec.LoadBalancerSourceRanges
		}

		// ExternalIPs
		obj.Spec.ExternalIPs = c.Ingress.Spec.ExternalIPs
		if len(obj.Spec.ExternalIPs) > 0 {
			sort.Strings(obj.Spec.ExternalIPs)
		}

		// opening other tcp ports
		mappings, _ := c.Ingress.PortMappings(c.cfg.CloudProvider)
		desiredPorts := make([]core.ServicePort, 0)
		for svcPort, target := range mappings {
			p := core.ServicePort{
				Name:       "tcp-" + strconv.Itoa(svcPort),
				Protocol:   "TCP",
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt(target.PodPort),
				NodePort:   int32(target.NodePort),
			}
			desiredPorts = append(desiredPorts, p)
		}
		obj.Spec.Ports = core_util.MergeServicePorts(obj.Spec.Ports, desiredPorts)

		// ExternalTrafficPolicy
		if c.Ingress.KeepSourceIP() {
			switch c.cfg.CloudProvider {
			case "gce", "gke", "azure", "acs", "aks", "metallb":
				// https://github.com/appscode/voyager/issues/276
				// ref: https://kubernetes.io/docs/tasks/services/source-ip/#source-ip-for-services-with-typeloadbalancer
				obj.Spec.ExternalTrafficPolicy = core.ServiceExternalTrafficPolicyTypeLocal
				if c.Ingress.HealthCheckNodeport() > 0 {
					obj.Spec.HealthCheckNodePort = int32(c.Ingress.HealthCheckNodeport())
				}
			}
		}

		// LoadBalancerIP
		switch c.cfg.CloudProvider {
		case "gce", "gke", "azure", "acs", "aks", "openstack", "metallb", "digitalocean", "linode":
			if ip := c.Ingress.LoadBalancerIP(); ip != nil {
				obj.Spec.LoadBalancerIP = ip.String()
			}
		}

		return obj
	})
}

func (c *loadBalancerController) ensurePods() (kutil.VerbType, error) {
	obj, err := wcs.NewObject(c.Ingress.WorkloadKind(), c.Ingress.OffshootName(), c.Ingress.Namespace)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	_, vt, err := c.WorkloadClient.Workloads(c.Ingress.Namespace).CreateOrPatch(obj, func(obj *wpi.Workload) *wpi.Workload {
		// deployment annotations
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Labels = c.Ingress.OffshootLabels()
		obj.ObjectMeta = c.ensureOwnerReference(obj.ObjectMeta)
		obj.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: c.Ingress.OffshootSelector(),
		}

		// assign number of replicas for initial creation only
		if obj.Spec.Replicas == nil {
			obj.Spec.Replicas = types.Int32P(c.Ingress.Replicas())
		}

		// pod annotations
		// delete last-applied-annotations, add new-annotations, store new-annotations keys
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = make(map[string]string)
		}
		lastAppliedKeys, _ := meta_util.GetStringValue(obj.Spec.Template.Annotations, api.LastAppliedAnnotationKeys)
		for _, key := range strings.Split(lastAppliedKeys, ",") {
			delete(obj.Spec.Template.Annotations, key)
		}
		newKeys := make([]string, 0)
		if ans, ok := c.Ingress.PodsAnnotations(); ok {
			for k, v := range ans {
				obj.Spec.Template.Annotations[k] = v
				newKeys = append(newKeys, k)
			}
		}
		sort.Strings(newKeys)
		obj.Spec.Template.Annotations[api.LastAppliedAnnotationKeys] = strings.Join(newKeys, ",")

		// pod spec
		obj.Spec.Template.ObjectMeta.Labels = c.Ingress.OffshootSelector()
		obj.Spec.Template.Spec.Affinity = c.Ingress.Spec.Affinity
		obj.Spec.Template.Spec.SchedulerName = c.Ingress.Spec.SchedulerName
		obj.Spec.Template.Spec.Tolerations = c.Ingress.Spec.Tolerations
		obj.Spec.Template.Spec.NodeSelector = c.Ingress.Spec.NodeSelector
		obj.Spec.Template.Spec.ImagePullSecrets = c.Ingress.Spec.ImagePullSecrets
		obj.Spec.Template.Spec.PriorityClassName = c.Ingress.Spec.PriorityClassName
		obj.Spec.Template.Spec.Priority = c.Ingress.Spec.Priority
		obj.Spec.Template.Spec.SecurityContext = c.Ingress.Spec.SecurityContext
		obj.Spec.Template.Spec.TerminationGracePeriodSeconds = c.Ingress.Spec.TerminationGracePeriodSeconds
		if c.cfg.EnableRBAC {
			obj.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()
		}

		// volume spec
		obj.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
			obj.Spec.Template.Spec.Volumes,
			core.Volume{
				Name: TLSCertificateVolumeName,
				VolumeSource: core.VolumeSource{
					EmptyDir: &core.EmptyDirVolumeSource{},
				},
			},
		)
		if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
			obj.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
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
				},
			)
		}
		for _, configVolume := range c.Ingress.Spec.ConfigVolumes {
			obj.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
				obj.Spec.Template.Spec.Volumes,
				core.Volume{
					Name: configVolume.Name,
					VolumeSource: core.VolumeSource{
						ConfigMap: configVolume.ConfigMap,
						Secret:    configVolume.Secret,
						Projected: configVolume.Projected,
					},
				},
			)
		}

		// container spec
		haproxyContainer := core.Container{
			Name:  "haproxy",
			Image: c.cfg.HAProxyImage,
			Args: append([]string{
				fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
				fmt.Sprintf("--burst=%v", c.cfg.Burst),
				fmt.Sprintf("--cloud-provider=%s", c.cfg.CloudProvider),
				fmt.Sprintf("--ingress-api-version=%s", c.Ingress.APISchema()),
				fmt.Sprintf("--ingress-name=%s", c.Ingress.Name),
				fmt.Sprintf("--qps=%v", c.cfg.QPS),
			}, cli.LoggerOptions.ToFlags()...),
			Env: c.ensureEnvVars([]core.EnvVar{
				{
					Name:  analytics.Key,
					Value: cli.AnalyticsClientID,
				},
			}),
			Ports:          []core.ContainerPort{},
			Resources:      c.Ingress.Spec.Resources,
			LivenessProbe:  c.Ingress.Spec.LivenessProbe,
			ReadinessProbe: c.Ingress.Spec.ReadinessProbe,
			VolumeMounts: []core.VolumeMount{
				{
					Name:      TLSCertificateVolumeName,
					MountPath: "/etc/ssl/private/haproxy",
				},
			},
		}
		if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
			haproxyContainer.VolumeMounts = append(
				haproxyContainer.VolumeMounts,
				core.VolumeMount{
					Name:      ErrorFilesVolumeName,
					MountPath: ErrorFilesLocation,
				},
			)
		}
		for _, configVolume := range c.Ingress.Spec.ConfigVolumes {
			haproxyContainer.VolumeMounts = append(
				haproxyContainer.VolumeMounts,
				core.VolumeMount{
					Name:      configVolume.Name,
					MountPath: configVolume.MountPath,
				},
			)
		}
		for _, podPort := range c.Ingress.PodPorts() {
			p := core.ContainerPort{
				Name:          "tcp-" + strconv.Itoa(podPort),
				Protocol:      "TCP",
				ContainerPort: int32(podPort),
			}
			haproxyContainer.Ports = append(haproxyContainer.Ports, p)
		}
		if c.Ingress.Stats() {
			haproxyContainer.Ports = append(haproxyContainer.Ports, core.ContainerPort{
				Name:          api.StatsPortName,
				Protocol:      "TCP",
				ContainerPort: int32(c.Ingress.StatsPort()),
			})
		}

		// upsert haproxy and exporter containers
		obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(obj.Spec.Template.Spec.Containers, haproxyContainer)
		if exporter, _ := c.getExporterSidecar(); exporter != nil {
			obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(obj.Spec.Template.Spec.Containers, *exporter)
		}

		return obj
	})
	return vt, err
}

func (c *loadBalancerController) updateStatus() error {
	var statuses []core.LoadBalancerIngress

	for i := 0; i < 50; i++ {
		time.Sleep(time.Second * 10)
		if svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{}); err == nil {
			if len(svc.Status.LoadBalancer.Ingress) >= 1 {
				statuses = svc.Status.LoadBalancer.Ingress
				break
			}
		}
	}

	if len(statuses) > 0 {
		if c.Ingress.APISchema() == api.APISchemaIngress {
			ing, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).Get(c.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return errors.WithStack(err)
			}
			ing.Status.LoadBalancer.Ingress = statuses
			_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).UpdateStatus(ing)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			ing, err := c.VoyagerClient.VoyagerV1beta1().Ingresses(c.Ingress.Namespace).Get(c.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return errors.WithStack(err)
			}
			_, err = util.UpdateIngressStatus(c.VoyagerClient.VoyagerV1beta1(), ing, func(in *api.IngressStatus) *api.IngressStatus {
				in.LoadBalancer.Ingress = statuses
				return in
			}, api.EnableStatusSubresource)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func ipnet(spec string) (string, bool) {
	spec = strings.TrimSpace(spec)
	_, ipnet, err := net.ParseCIDR(spec)
	if err != nil {
		return "", false
	}
	return ipnet.String(), true
}
