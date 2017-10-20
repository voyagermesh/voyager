package voyager

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ResourceKindIngress = "Ingress"
	ResourceNameIngress = "ingress"
	ResourceTypeIngress = "ingresses"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Custom Ingress type for Voyager.
type Ingress struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the Ingress.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Spec IngressSpec `json:"spec,omitempty"`

	// Status is the current state of the Ingress.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Status IngressStatus `json:"status,omitempty"`
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
	Resources apiv1.ResourceRequirements `json:"resources,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *apiv1.Affinity `json:"affinity,omitempty" protobuf:"bytes,18,opt,name=affinity"`

	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,19,opt,name=schedulerName"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []apiv1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,22,opt,name=tolerations"`
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
	// LoadBalancer contains the current status of the load-balancer.
	LoadBalancer apiv1.LoadBalancerStatus `json:"loadBalancer,omitempty"`
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
	// default backend, is left to the controller fulfilling the Ingress. Http is
	// currently the only supported IngressRuleValue.
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
	// port to listen http(s) connections.
	Port intstr.IntOrString `json:"port,omitempty"`

	// Set noTLS = true to force plain text. Else, auto detect like present
	NoTLS bool `json:"noTLS,omitempty"`

	// Specifies the node port of the referenced service.
	NodePort intstr.IntOrString `json:"nodePort,omitempty"`

	// A collection of paths that map requests to backends.
	Paths []HTTPIngressPath `json:"paths"`
}

type TCPIngressRuleValue struct {
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
	ALPN []string `json:"alpn,omitempty"`
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
	// Host names to forward traffic to. If empty traffic will be
	// forwarded to all subsets instance.
	// If set only matched hosts will get the traffic.
	// This is an handy way to send traffic to Specific
	// StatefulSet pod.
	// IE. Setting [web-0] will send traffic to only web-0 host
	// for this StatefulSet, https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#creating-a-statefulset
	HostNames []string `json:"hostNames,omitempty"`

	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName,omitempty"`

	// Specifies the port of the referenced service.
	ServicePort intstr.IntOrString `json:"servicePort,omitempty"`

	// Serialized HAProxy rules to apply on server backend including
	// request, response or header rewrite. acls also can be used.
	// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#1
	BackendRule []string `json:"backendRule,omitempty"`
}

type HTTPIngressBackend struct {
	IngressBackend `json:",inline,omitempty"`

	// Path rewrite rules with haproxy formatted regex.
	//
	// Deprecated: Use backendRule, will be removed.
	RewriteRule []string `json:"rewriteRule,omitempty"`

	// Header rules to modifies the header.
	//
	// Deprecated: Use backendRule, will be removed.
	HeaderRule []string `json:"headerRule,omitempty"`
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
