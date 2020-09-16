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

package v1

import (
	"strings"

	kutil "kmodules.xyz/client-go"

	core "k8s.io/api/core/v1"
)

type AgentType string

const (
	KeyAgent   = "monitoring.appscode.com/agent"
	KeyService = "monitoring.appscode.com/service"

	VendorPrometheus = "prometheus.io"

	AgentPrometheus         AgentType = VendorPrometheus
	AgentPrometheusBuiltin  AgentType = VendorPrometheus + "/builtin"
	AgentPrometheusOperator AgentType = VendorPrometheus + "/operator"
	// Deprecated
	AgentCoreOSPrometheus AgentType = VendorPrometheus + "/coreos-operator"
	// Deprecated
	DeprecatedAgentCoreOSPrometheus AgentType = "coreos-prometheus-operator"
)

func (at AgentType) Vendor() string {
	if at == DeprecatedAgentCoreOSPrometheus {
		return VendorPrometheus
	}
	return strings.SplitN(string(at), "/", 2)[0]
}

type AgentSpec struct {
	Agent      AgentType       `json:"agent,omitempty" protobuf:"bytes,1,opt,name=agent,casttype=AgentType"`
	Prometheus *PrometheusSpec `json:"prometheus,omitempty" protobuf:"bytes,2,opt,name=prometheus"`

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// Deprecated: use prometheus.exporter.args
	Args []string `json:"args,omitempty" protobuf:"bytes,3,rep,name=args"`

	// List of environment variables to set in the container.
	// Cannot be updated.
	// Deprecated
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// Deprecated: use prometheus.exporter.env
	Env []core.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=env"`

	// Compute Resources required by exporter container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	// Deprecated: use prometheus.exporter.resources
	Resources core.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,5,opt,name=resources"`

	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	// Deprecated: use prometheus.exporter.securityContext
	SecurityContext *core.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,6,opt,name=securityContext"`
}

type PrometheusSpec struct {
	// Port number for the exporter side car.
	// Deprecated: use exporter.port
	Port int32 `json:"port,omitempty" protobuf:"varint,1,opt,name=port"`

	// Labels are key value pairs that is used to select Prometheus instance via ServiceMonitor labels.
	// +optional
	// Deprecated: use prometheus.serviceMonitor.labels
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`

	// Interval at which metrics should be scraped
	// +optional
	// Deprecated: use prometheus.serviceMonitor.interval
	Interval string `json:"interval,omitempty" protobuf:"bytes,3,opt,name=interval"`

	ServiceMonitor *ServiceMonitorSpec `json:"serviceMonitor,omitempty" protobuf:"bytes,4,opt,name=serviceMonitor"`

	Exporter *PrometheusExporterSpec `json:"exporter,omitempty" protobuf:"bytes,5,opt,name=exporter"`
}

type ServiceMonitorSpec struct {
	// Labels are key value pairs that is used to select Prometheus instance via ServiceMonitor labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,1,rep,name=labels"`

	// Interval at which metrics should be scraped
	// +optional
	Interval string `json:"interval,omitempty" protobuf:"bytes,2,opt,name=interval"`
}

type PrometheusExporterSpec struct {
	// Port number for the exporter side car.
	Port int32 `json:"port,omitempty" protobuf:"varint,1,opt,name=port"`

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`

	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []core.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,3,rep,name=env"`

	// Compute Resources required by exporter container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources core.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,4,opt,name=resources"`

	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *core.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
}

type Agent interface {
	GetType() AgentType
	CreateOrUpdate(sp StatsAccessor, spec *AgentSpec) (kutil.VerbType, error)
	Delete(sp StatsAccessor) (kutil.VerbType, error)
}

type StatsAccessor interface {
	GetNamespace() string
	ServiceName() string
	ServiceMonitorName() string
	ServiceMonitorAdditionalLabels() map[string]string
	Path() string
	// Scheme is used to determine url scheme /metrics endpoint
	Scheme() string
}
