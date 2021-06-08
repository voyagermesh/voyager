/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	"voyagermesh.dev/voyager/pkg/config"
	"voyagermesh.dev/voyager/pkg/eventer"
	"voyagermesh.dev/voyager/third_party/forked/cloudprovider"
	_ "voyagermesh.dev/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "voyagermesh.dev/voyager/third_party/forked/cloudprovider/providers/fake"

	"github.com/pkg/errors"
	pcm "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"gomodules.xyz/flags"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/analytics"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/monitoring-agent-api/parser"
	wpi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type hostPortController struct {
	*controller
	CloudManager cloudprovider.Interface
}

var _ Controller = &hostPortController{}

func NewHostPortController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	workloadClient wcs.Interface,
	crdClient crd_cs.Interface,
	extClient cs.Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	cfg config.Config,
	ingress *api.Ingress,
	recorder record.EventRecorder) Controller {
	c := &hostPortController{
		controller: &controller{
			logger:          klogr.New().WithName("hostport").WithValues("namespace", ingress.Namespace, "name", ingress.Name),
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
	c.logger.Info("Initializing cloud manager", "provider", cfg.CloudProvider)
	if cfg.CloudProvider == api.ProviderAWS || cfg.CloudProvider == api.ProviderGCE || cfg.CloudProvider == api.ProviderAzure {
		cloudInterface, err := cloudprovider.InitCloudProvider(cfg.CloudProvider, cfg.CloudConfigFile)
		if err != nil {
			c.logger.Error(err, "Failed to initialize cloud provider", cfg.CloudProvider)
		} else {
			c.logger.Info("Initialized cloud manager", "provider", cfg.CloudProvider)
			c.CloudManager = cloudInterface
		}
	} else if cfg.CloudProvider == api.ProviderGKE {
		cloudInterface, err := cloudprovider.InitCloudProvider(api.ProviderGCE, cfg.CloudConfigFile)
		if err != nil {
			c.logger.Error(err, "Failed to initialize cloud provider", "provider", cfg.CloudProvider)
		} else {
			c.logger.Info("Initialized cloud provider", "provider", cfg.CloudProvider)
			c.CloudManager = cloudInterface
		}
	} else if cfg.CloudProvider == api.ProviderAzure {
		cloudInterface, err := cloudprovider.InitCloudProvider(api.ProviderAzure, cfg.CloudConfigFile)
		if err != nil {
			c.logger.Error(err, "Failed to initialize cloud provider", "provider", cfg.CloudProvider)
		} else {
			c.logger.Info("Initialized cloud provider", "provider", cfg.CloudProvider)
			c.CloudManager = cloudInterface
		}
	} else if cfg.CloudProvider == api.ProviderMinikube {
		c.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		c.logger.Info("No cloud manager found", "provider", cfg.CloudProvider)
	}
	return c
}

func (c *hostPortController) Reconcile() error {
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

	// Ensure service account
	err := c.reconcileRBAC()
	if err != nil {
		return err
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

	if svc, vt, err := c.ensureService(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressServiceReconcileFailed,
			"Failed to reconcile headless Service %s, Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return errors.WithStack(err)
	} else if err = c.EnsureFirewall(svc); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressFirewallUpdateFailed,
			"Failed to reconcile firewall. Reason: %v",
			err,
		)
		return errors.WithStack(err)
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressServiceReconcileSuccessful,
			"Successfully %s headless Service %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

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
			klog.Warningf("failed to delete stats Service %s, reason: %s", c.Ingress.StatsServiceName(), err)
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
			klog.Warningf("failed to delete old monitoring agent, reason: %s", err)
		}
	}

	return nil
}

func (c *hostPortController) FirewallSupported() bool {
	if c.CloudManager != nil {
		_, ok := c.CloudManager.Firewall()
		return ok
	}
	return false
}

func (c *hostPortController) EnsureFirewall(svc *core.Service) error {
	if c.CloudManager != nil {
		if fw, ok := c.CloudManager.Firewall(); ok {
			nodes, err := c.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(c.Ingress.Spec.NodeSelector).String(),
			})
			if err != nil {
				return err
			}
			hostnames := make([]string, len(nodes.Items))
			for i, node := range nodes.Items {
				hostnames[i] = node.Name
			}
			err = fw.EnsureFirewall(context.Background(), svc, hostnames)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// make sure all delete calls require only ingress name and namespace
func (c *hostPortController) Delete() {
	if err := c.deletePods(); err != nil {
		c.logger.Error(err, "failed to delete pods")
	}
	if err := c.deleteConfigMap(); err != nil {
		c.logger.Error(err, "failed to delete configmap")
	}
	if err := c.ensureRBACDeleted(); err != nil {
		c.logger.Error(err, "failed to delete rbac")
	}
	if err := c.ensureServiceDeleted(); err != nil {
		c.logger.Error(err, "failed to delete service")
	}
	if c.CloudManager != nil { // TODO @ Dipta: fix this
		if fw, ok := c.CloudManager.Firewall(); ok {
			err := fw.EnsureFirewallDeleted(context.Background(), &core.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Ingress.OffshootName(),
					Namespace: c.Ingress.Namespace,
				},
			})
			if err != nil {
				c.logger.Error(err, "failed to delete firewall")
			}
		}
	}
	// delete agent before deleting stat service
	if err := c.ensureMonitoringAgentDeleted(nil); err != nil {
		c.logger.Error(err, "failed to delete monitoring agent")
	}
	if err := c.ensureStatsServiceDeleted(); err != nil {
		c.logger.Error(err, "failed to delete stats service")
	}
}

// Create a Headless service without selectors
// We just want kubernetes to assign a stable UID to the service. This is used inside EnsureFirewall()
func (c *hostPortController) ensureService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchService(context.TODO(), c.KubeClient, meta, func(in *core.Service) *core.Service {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))
		in.Spec.Type = core.ServiceTypeClusterIP
		in.Spec.Selector = c.Ingress.OffshootSelector()
		in.Spec.ClusterIP = "None"

		in.Labels = c.Ingress.OffshootLabels()
		// Annotations
		if in.Annotations == nil {
			in.Annotations = make(map[string]string)
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		// delete last applied ServiceAnnotations
		// add new ServiceAnnotations
		// store new ServiceAnnotations keys
		lastAppliedKeys, _ := meta_util.GetStringValue(in.Annotations, api.LastAppliedAnnotationKeys)
		for _, key := range strings.Split(lastAppliedKeys, ",") {
			delete(in.Annotations, key)
		}
		newKeys := make([]string, 0)
		if ans, ok := c.Ingress.ServiceAnnotations(c.cfg.CloudProvider); ok {
			for k, v := range ans {
				in.Annotations[k] = v
				newKeys = append(newKeys, k)
			}
		}
		sort.Strings(newKeys)
		in.Annotations[api.LastAppliedAnnotationKeys] = strings.Join(newKeys, ",")

		// ExternalIPs
		in.Spec.ExternalIPs = c.Ingress.Spec.ExternalIPs
		if len(in.Spec.ExternalIPs) > 0 {
			sort.Strings(in.Spec.ExternalIPs)
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
			}
			desiredPorts = append(desiredPorts, p)
		}
		in.Spec.Ports = core_util.MergeServicePorts(in.Spec.Ports, desiredPorts)

		return in
	}, metav1.PatchOptions{})
}

func (c *hostPortController) ensurePods() (kutil.VerbType, error) {
	obj, err := wcs.NewObject(c.Ingress.WorkloadKind(), c.Ingress.OffshootName(), c.Ingress.Namespace)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	_, vt, err := c.WorkloadClient.Workloads(c.Ingress.Namespace).CreateOrPatch(context.TODO(), obj, func(in *wpi.Workload) *wpi.Workload {
		// deployment annotations
		if in.Annotations == nil {
			in.Annotations = make(map[string]string)
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Labels = c.Ingress.OffshootLabels()
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))
		in.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: c.Ingress.OffshootSelector(),
		}

		// assign number of replicas only when there's no controlling hpa
		if in.Spec.Replicas == nil || !c.isHPAControlled() {
			in.Spec.Replicas = pointer.Int32P(c.Ingress.Replicas())
		}

		// pod annotations
		// delete last-applied-annotations, add new-annotations, store new-annotations keys
		if in.Spec.Template.Annotations == nil {
			in.Spec.Template.Annotations = make(map[string]string)
		}
		lastAppliedKeys, _ := meta_util.GetStringValue(in.Spec.Template.Annotations, api.LastAppliedAnnotationKeys)
		for _, key := range strings.Split(lastAppliedKeys, ",") {
			delete(in.Spec.Template.Annotations, key)
		}
		newKeys := make([]string, 0)
		if ans, ok := c.Ingress.PodsAnnotations(); ok {
			for k, v := range ans {
				in.Spec.Template.Annotations[k] = v
				newKeys = append(newKeys, k)
			}
		}
		sort.Strings(newKeys)
		in.Spec.Template.Annotations[api.LastAppliedAnnotationKeys] = strings.Join(newKeys, ",")

		// pod spec
		in.Spec.Template.ObjectMeta.Labels = c.Ingress.OffshootSelector()
		in.Spec.Template.Spec.Affinity = c.Ingress.Spec.Affinity
		in.Spec.Template.Spec.SchedulerName = c.Ingress.Spec.SchedulerName
		in.Spec.Template.Spec.Tolerations = c.Ingress.Spec.Tolerations
		in.Spec.Template.Spec.NodeSelector = c.Ingress.Spec.NodeSelector
		in.Spec.Template.Spec.ImagePullSecrets = c.Ingress.Spec.ImagePullSecrets
		in.Spec.Template.Spec.PriorityClassName = c.Ingress.Spec.PriorityClassName
		in.Spec.Template.Spec.Priority = c.Ingress.Spec.Priority
		in.Spec.Template.Spec.SecurityContext = c.Ingress.Spec.SecurityContext
		in.Spec.Template.Spec.HostNetwork = true
		in.Spec.Template.Spec.DNSPolicy = core.DNSClusterFirstWithHostNet
		in.Spec.Template.Spec.TerminationGracePeriodSeconds = c.Ingress.Spec.TerminationGracePeriodSeconds
		in.Spec.Template.Spec.ServiceAccountName = c.Ingress.OffshootName()

		// volume spec
		in.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
			in.Spec.Template.Spec.Volumes,
			core.Volume{
				Name: TLSCertificateVolumeName,
				VolumeSource: core.VolumeSource{
					EmptyDir: &core.EmptyDirVolumeSource{},
				},
			},
		)
		if len(c.Ingress.ErrorFilesConfigMapName()) > 0 {
			in.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
				in.Spec.Template.Spec.Volumes,
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
			in.Spec.Template.Spec.Volumes = core_util.UpsertVolume(
				in.Spec.Template.Spec.Volumes,
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
			}, flags.LoggerOptions.ToFlags()...),
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
				HostPort:      int32(podPort),
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
		in.Spec.Template.Spec.Containers = core_util.UpsertContainer(in.Spec.Template.Spec.Containers, haproxyContainer)
		if exporter, _ := c.getExporterSidecar(); exporter != nil {
			in.Spec.Template.Spec.Containers = core_util.UpsertContainer(in.Spec.Template.Spec.Containers, *exporter)
		}

		return in
	}, metav1.PatchOptions{})
	return vt, err
}
