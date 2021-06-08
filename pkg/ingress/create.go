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

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	"gomodules.xyz/flags"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/analytics"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/monitoring-agent-api/parser"
)

const (
	TLSCertificateVolumeName = "voyager-certdir"
	ErrorFilesVolumeName     = "voyager-errorfiles"
	ErrorFilesLocation       = "/srv/voyager/errorfiles"
	ErrorFilesCommand        = "errorfile"
)

func (c *controller) ensureConfigMap() (*core.ConfigMap, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.OffshootName(),
		Namespace: c.Ingress.Namespace,
	}
	return core_util.CreateOrPatchConfigMap(context.TODO(), c.KubeClient, meta, func(in *core.ConfigMap) *core.ConfigMap {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))
		if in.Annotations == nil {
			in.Annotations = make(map[string]string)
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()
		in.Data = map[string]string{
			"haproxy.cfg": c.HAProxyConfig,
		}
		return in
	}, metav1.PatchOptions{})
}

func (c *controller) getExporterSidecar() (*core.Container, error) {
	if !c.Ingress.Stats() {
		return nil, nil // Don't add sidecar is stats is not exposed.
	}
	monSpec, err := parser.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
	if err != nil {
		return nil, err
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		return &core.Container{
			Name: "exporter",
			Args: append([]string{
				"export",
				fmt.Sprintf("--address=:%d", monSpec.Prometheus.Exporter.Port),
				fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
			}, flags.LoggerOptions.ToFlags()...),
			Env: []core.EnvVar{
				{
					Name:  analytics.Key,
					Value: cli.AnalyticsClientID,
				},
			},
			Image:           c.cfg.ExporterImage,
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          api.ExporterPortName,
					Protocol:      core.ProtocolTCP,
					ContainerPort: int32(monSpec.Prometheus.Exporter.Port),
				},
			},
		}, nil
	}
	return nil, nil
}

func (c *controller) ensureStatsService() (*core.Service, kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      c.Ingress.StatsServiceName(),
		Namespace: c.Ingress.Namespace,
	}

	return core_util.CreateOrPatchService(context.TODO(), c.KubeClient, meta, func(in *core.Service) *core.Service {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))
		in.Labels = c.Ingress.StatsLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Spec.Selector = c.Ingress.OffshootSelector()

		desired := []core.ServicePort{
			{
				Name:       api.StatsPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(c.Ingress.StatsPort()),
				TargetPort: intstr.FromString(api.StatsPortName),
			},
		}
		monSpec, err := parser.Parse(c.Ingress.Annotations, api.EngressKey, api.DefaultExporterPortNumber)
		if err == nil && monSpec != nil && monSpec.Prometheus != nil {
			desired = append(desired, core.ServicePort{
				Name:       api.ExporterPortName,
				Protocol:   core.ProtocolTCP,
				Port:       int32(monSpec.Prometheus.Exporter.Port),
				TargetPort: intstr.FromString(api.ExporterPortName),
			})
		}
		in.Spec.Ports = core_util.MergeServicePorts(in.Spec.Ports, desired)
		return in
	}, metav1.PatchOptions{})
}
