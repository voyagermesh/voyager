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

package v1beta1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ResourceKindIngress = "Ingress"
	ResourceIngress     = "ingress"
	ResourceIngresses   = "ingresses"
)

// Ingress is a collection of rules that allow inbound connections to reach the
// endpoints defined by a backend. An Ingress can be configured to give services
// externally-reachable urls, load balance traffic, terminate SSL, offer name
// based virtual hosting etc.

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ingresses,singular=ingress,shortName=ing,categories={networking,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hosts",type="string",JSONPath=".spec.rules[0].host"
// +kubebuilder:printcolumn:name="LOAD_BALANCER_IP",type="string",JSONPath=".status.loadBalancer.ingress"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Ingress struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IngressSpec   `json:"spec,omitempty"`
	Status            IngressStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressList is a collection of Ingress.
type IngressList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Ingress.
	Items []Ingress `json:"items"`
}

// IngressSpec describes the Ingress the user wishes to exist.
type IngressSpec struct {
	// A default backend capable of servicing requests that don't match any
	// rule. At least one of 'backend' or 'rules' must be specified. This field
	// is optional to allow the loadbalancer controller or defaulting logic to
	// specify a global default.
	Backend *HTTPIngressBackend `json:"backend,omitempty"`

	// TLS is the TLS configuration. Currently the Ingress only supports a single TLS
	// port, 443, and assumes TLS termination. If multiple members of this
	// list specify different hosts, they will be multiplexed on the same
	// port according to the hostname specified through the SNI TLS extension.
	TLS []IngressTLS `json:"tls,omitempty"`

	// Config volumes are used to mount any secret or configmap into HAProxy pods.
	ConfigVolumes []VolumeSource `json:"configVolumes,omitempty"`

	// Frontend rules specifies a set of rules that should be applied in
	// HAProxy frontend configuration. The set of keywords are from here
	// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.1
	// Only frontend sections can be applied here. It is up to user to
	// provide valid set of rules.
	// This allows acls or other options in frontend sections in HAProxy config.
	// Frontend rules will be mapped with Ingress Rules according to port.
	FrontendRules []FrontendRule `json:"frontendRules,omitempty"`

	// A list of host rules used to configure the Ingress. If unspecified, or
	// no rule matches, all traffic is sent to the default backend.
	Rules []IngressRule `json:"rules,omitempty"`

	// Optional: If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs. This field will be ignored if the
	// cloud-provider does not support the feature.
	// https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`

	// Compute Resources required by the sidecar container.
	Resources core.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *core.Affinity `json:"affinity,omitempty"`

	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []core.Toleration `json:"tolerations,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// If specified, indicates the pod's priority. "system-node-critical" and
	// "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// The priority value. Various system components use this field to find the
	// priority of the pod. When Priority Admission Controller is enabled, it
	// prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *core.PodSecurityContext `json:"securityContext,omitempty"`

	// externalIPs is a list of IP addresses for which nodes in the cluster
	// will also accept traffic for this service.  These IPs are not managed by
	// Kubernetes.  The user is responsible for ensuring that traffic arrives
	// at a node with this IP.  A common example is external load-balancers
	// that are not part of the Kubernetes system.
	// +optional
	ExternalIPs []string `json:"externalIPs,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	// +optional
	LivenessProbe *core.Probe `json:"livenessProbe,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// +optional
	ReadinessProbe *core.Probe `json:"readinessProbe,omitempty"`

	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 30 seconds.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// IngressTLS describes the transport layer security associated with an Ingress.
type IngressTLS struct {
	// Hosts are a list of hosts included in the TLS certificate. The values in
	// this list must match the name/s used in the tlsSecret. Defaults to the
	// wildcard host setting for the loadbalancer controller fulfilling this
	// Ingress, if left unspecified.
	Hosts []string `json:"hosts,omitempty"`
	// SecretName is the name of the secret used to terminate SSL traffic on 443.
	// Field is left optional to allow SSL routing based on SNI hostname alone.
	// If the SNI host in a listener conflicts with the "Host" header field used
	// by an IngressRule, the SNI host is used for termination and value of the
	// Host header is used for routing.
	// Deprecated
	SecretName string `json:"secretName,omitempty"`

	// Ref to used tls termination.
	Ref *LocalTypedReference `json:"ref,omitempty"`
}

// IngressStatus describe the current state of the Ingress.
type IngressStatus struct {
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LoadBalancer contains the current status of the load-balancer.
	LoadBalancer core.LoadBalancerStatus `json:"loadBalancer,omitempty"`
}

// IngressRule represents the rules mapping the paths under a specified host to
// the related backend services. Incoming requests are first evaluated for a host
// match, then routed to the backend associated with the matching IngressRuleValue.
type IngressRule struct {
	// Host is the fully qualified domain name of a network host, as defined
	// by RFC 3986. Note the following deviations from the "host" part of the
	// URI as defined in the RFC:
	// 1. IPs are not allowed. Currently an IngressRuleValue can only apply to the
	//	  IP in the Spec of the parent Ingress.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//	  Currently the port of an Ingress is implicitly :80 for http and
	//	  :443 for https.
	// Both these may change in the future.
	// Incoming requests are matched against the host before the IngressRuleValue.
	// If the host is unspecified, the Ingress routes all traffic based on the
	// specified IngressRuleValue.
	Host string `json:"host,omitempty"`
	// IngressRuleValue represents a rule to route requests for this IngressRule.
	// If unspecified, the rule defaults to a http catch-all. Whether that sends
	// just traffic matching the host to the default backend or all traffic to the
	// default backend, is left to the controller fulfilling the Ingress. Http and TCP are
	// currently supported IngressRuleValue.
	IngressRuleValue `json:",inline,omitempty"`
}

// IngressRuleValue represents a rule to apply against incoming requests. If the
// rule is satisfied, the request is routed to the specified backend. Currently
// mixing different types of rules in a single Ingress is disallowed, so exactly
// one of the following must be set.
type IngressRuleValue struct {
	//TODO:
	// 1. Consider renaming this resource and the associated rules so they
	// aren't tied to Ingress. They can be used to route intra-cluster traffic.
	// 2. Consider adding fields for Ingress-type specific global options
	// usable by a loadbalancer, like http keep-alive.

	HTTP *HTTPIngressRuleValue `json:"http,omitempty"`

	TCP *TCPIngressRuleValue `json:"tcp,omitempty"`
}

// HTTPIngressRuleValue is a list of http selectors pointing to backends.
// In the example: http://<host>/<path>?<searchpart> -> backend where
// where parts of the url correspond to RFC 3986, this resource will be used
// to match against everything after the last '/' and before the first '?'
// or '#'.
type HTTPIngressRuleValue struct {
	// The network address to listen HTTP(s) connections on.
	Address string `json:"address,omitempty"`

	// port to listen http(s) connections.
	Port intstr.IntOrString `json:"port,omitempty"`

	// Set noTLS = true to force plain text. Else, auto detect like present
	NoTLS bool `json:"noTLS,omitempty"`

	// Specifies the node port of the referenced service.
	NodePort intstr.IntOrString `json:"nodePort,omitempty"`

	// Application-Layer Protocol Negotiation (ALPN) is a Transport Layer Security (TLS)
	// extension for application layer protocol negotiation. ALPN allows the application
	// layer to negotiate which protocol should be performed over a secure connection in a
	// manner which avoids additional round trips and which is independent of the application
	// layer protocols. It is used by HTTP/2.
	// If provided a list of alpn will be added to port as alpn option1,option2,...
	// If SecretName is Provided this secret will be used to terminate SSL with alpn options.
	// If Secret name is not provided backend server is responsible for handling SSL.
	// Note that, the order of the options indicates the preference
	// If the ALPN list contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#option%20http-use-htx
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#alpn
	ALPN []string `json:"alpn,omitempty"`

	// HTTP protocol to use
	// If the Proto contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://www.haproxy.com/blog/haproxy-1-9-2-adds-grpc-support/
	Proto string `json:"proto,omitempty"`

	// A collection of paths that map requests to backends.
	Paths []HTTPIngressPath `json:"paths"`
}

type TCPIngressRuleValue struct {
	// The network address to listen TCP connections on.
	Address string `json:"address,omitempty"`

	// port to listen tcp connections.
	Port intstr.IntOrString `json:"port,omitempty"`

	// Set noTLS = true to force plain text. Else, auto detect like present
	NoTLS bool `json:"noTLS,omitempty"`

	// Specifies the node port of the referenced service.
	NodePort intstr.IntOrString `json:"nodePort,omitempty"`

	// Backend to forward the requests.
	Backend IngressBackend `json:"backend,omitempty"`

	// Application-Layer Protocol Negotiation (ALPN) is a Transport Layer Security (TLS)
	// extension for application layer protocol negotiation. ALPN allows the application
	// layer to negotiate which protocol should be performed over a secure connection in a
	// manner which avoids additional round trips and which is independent of the application
	// layer protocols. It is used by HTTP/2.
	// If provided a list of alpn will be added to port as alpn option1,option2,...
	// If SecretName is Provided this secret will be used to terminate SSL with alpn options.
	// If Secret name is not provided backend server is responsible for handling SSL.
	// Note that, the order of the options indicates the preference
	// If the ALPN list contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#option%20http-use-htx
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#alpn
	ALPN []string `json:"alpn,omitempty"`

	// HTTP protocol to use
	// If the Proto contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://www.haproxy.com/blog/haproxy-1-9-2-adds-grpc-support/
	Proto string `json:"proto,omitempty"`
}

// HTTPIngressPath associates a path regex with a backend. Incoming urls matching
// the path are forwarded to the backend.
type HTTPIngressPath struct {
	// Path is a extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	Path string `json:"path,omitempty"`

	// Backend defines the referenced service endpoint to which the traffic
	// will be forwarded to.
	Backend HTTPIngressBackend `json:"backend,omitempty"`
}

// IngressBackend describes all endpoints for a given service and port.
type IngressBackend struct {
	// User can specify backend name for using it with custom acl
	// Otherwise it will be generated
	Name string `json:"name,omitempty"`

	// Host names to forward traffic to. If empty traffic will be
	// forwarded to all subsets instance.
	// If set only matched hosts will get the traffic.
	// This is an handy way to send traffic to Specific
	// StatefulSet pod.
	// IE. Setting [web-0] will send traffic to only web-0 host
	// for this StatefulSet, https://kubernetes.io/docs/tasks/stateful-application/basic-stateful-set/#creating-a-statefulset
	HostNames []string `json:"hostNames,omitempty"`

	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName,omitempty"`

	// Specifies the port of the referenced service.
	ServicePort intstr.IntOrString `json:"servicePort,omitempty"`

	// Serialized HAProxy rules to apply on server backend including
	// request, response or header rewrite. acls also can be used.
	// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#1
	BackendRules []string `json:"backendRules,omitempty"`

	// Application-Layer Protocol Negotiation (ALPN) is a Transport Layer Security (TLS)
	// extension for application layer protocol negotiation. ALPN allows the application
	// layer to negotiate which protocol should be performed over a secure connection in a
	// manner which avoids additional round trips and which is independent of the application
	// layer protocols. It is used by HTTP/2.
	// If provided a list of alpn will be added to port as alpn option1,option2,...
	// If SecretName is Provided this secret will be used to terminate SSL with alpn options.
	// If Secret name is not provided backend server is responsible for handling SSL.
	// Note that, the order of the options indicates the preference
	// If the ALPN list contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#option%20http-use-htx
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#alpn
	ALPN []string `json:"alpn,omitempty"`

	// HTTP protocol to use
	// If the Proto contains "h2",  "option http-use-htx" will be added to enable HTX mode
	// https://www.haproxy.com/blog/haproxy-1-9-2-adds-grpc-support/
	Proto string `json:"proto,omitempty"`

	// Define the load balancing algorithm to be used in a backend.
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#balance
	LoadBalanceOn string `json:"loadBalanceOn,omitempty"`
}

type HTTPIngressBackend struct {
	IngressBackend `json:",inline,omitempty"`

	// Path rewrite rules with haproxy formatted regex.
	//
	// Deprecated: Use backendRule, will be removed.
	RewriteRules []string `json:"rewriteRules,omitempty"`

	// Header rules to modifies the header.
	//
	// Deprecated: Use backendRule, will be removed.
	HeaderRules []string `json:"headerRules,omitempty"`
}

type IngressRef struct {
	APISchema string `json:"apiSchema"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type FrontendRule struct {
	// Port indicates the frontend port where HAProxy is listening for connection
	Port intstr.IntOrString `json:"port,omitempty"`
	// Serialized rules
	Rules []string `json:"rules,omitempty"`

	Auth *AuthOption `json:"auth,omitempty"`
}

type AuthOption struct {
	Basic *BasicAuth `json:"basic,omitempty"`
	TLS   *TLSAuth   `json:"tls,omitempty"`
	OAuth []OAuth    `json:"oauth,omitempty"`
}

type OAuth struct {
	Host        string   `json:"host,omitempty"`
	AuthBackend string   `json:"authBackend,omitempty"`
	AuthPath    string   `json:"authPath,omitempty"`
	SigninPath  string   `json:"signinPath,omitempty"`
	Paths       []string `json:"paths,omitempty"`
}

type BasicAuth struct {
	SecretName string `json:"secretName,omitempty"`
	Realm      string `json:"realm,omitempty"`
}

type TLSAuthVerifyOption string

const (
	TLSAuthVerifyOptional TLSAuthVerifyOption = "optional"
	TLSAuthVerifyRequired TLSAuthVerifyOption = "required"
)

type TLSAuth struct {
	SecretName   string              `json:"secretName,omitempty"`
	VerifyClient TLSAuthVerifyOption `json:"verifyClient,omitempty"`
	Headers      map[string]string   `json:"headers,omitempty"`
	ErrorPage    string              `json:"errorPage,omitempty"`
}
