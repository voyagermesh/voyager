package api

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	EngressKey = "ingress.appscode.com"
	APISchema  = EngressKey + "/" + "api-schema" // APISchema = {APIGroup}/{APIVersion}

	APISchemaEngress = "appscode.com/v1beta1"
	APISchemaIngress = "extension/v1beta1"

	VoyagerPrefix = "voyager-"

	StickySession = EngressKey + "/" + "sticky-session"

	// LB stats options
	StatsOn          = EngressKey + "/" + "stats"
	StatsPort        = EngressKey + "/" + "stats-port"
	StatsSecret      = EngressKey + "/" + "stats-secret-name"
	StatsServiceName = EngressKey + "/" + "stats-service-name"
	DefaultStatsPort = 1936

	// Daemon, Persistent, LoadBalancer
	LBType = EngressKey + "/" + "type"

	LBTypeNodePort = "NodePort"
	LBTypeHostPort = "HostPort"
	// Deprecated, use LBTypeHostPort
	LBTypeDaemon       = "Daemon"
	LBTypeLoadBalancer = "LoadBalancer" // default

	// Runs HAProxy on a specific set of a hosts.
	NodeSelector = EngressKey + "/" + "node-selector"

	// Replicas specify # of HAProxy pods run (default 1)
	Replicas = EngressKey + "/" + "replicas"

	// LoadBalancer mode exposes HAProxy via a type=LoadBalancer service. This is the original version implemented by @sadlil
	// Uses nodeport and Cloud LoadBalancer exists beyond single HAProxy run
	LoadBalancerPersist = EngressKey + "/" + "persist" // "" or IP or non-empty

	// BackendWeight is the weight value of a Pod that was
	// addressed by the Endpoint, this weight will be added to server backend.
	// Traffic will be forwarded according to there weight.
	BackendWeight = EngressKey + "/" + "backend-weight"

	// https://github.com/appscode/voyager/issues/103
	// ServiceAnnotations is user provided annotations map that will be
	// applied to the service of that LoadBalancer.
	// ex: "ingress.appscode.com/service.annotation": {"key": "val"}
	ServiceAnnotations = EngressKey + "/" + "annotations-service"

	// PodAnnotations is user provided annotations map that will be
	// applied to the Pods (Deployment/ DaemonSet) of that LoadBalancer.
	// ex: "ingress.appscode.com/service.annotation": {"key": "val"}
	PodAnnotations = EngressKey + "/" + "annotations-pod"

	// Preserves source IP for LoadBalancer type ingresses. The actual configuration
	// generated depends on the underlying cloud provider.
	//
	//  - gce, gke, azure: Adds annotation service.beta.kubernetes.io/external-traffic: OnlyLocal
	// to services used to expose HAProxy.
	// ref: https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-typeloadbalancer
	//
	// - aws: Enforces the use of the PROXY protocol over any connection accepted by any of
	// the sockets declared on the same line. Versions 1 and 2 of the PROXY protocol
	// are supported and correctly detected. The PROXY protocol dictates the layer
	// 3/4 addresses of the incoming connection to be used everywhere an address is
	// used, with the only exception of "tcp-request connection" rules which will
	// only see the real connection address. Logs will reflect the addresses
	// indicated in the protocol, unless it is violated, in which case the real
	// address will still be used.  This keyword combined with support from external
	// components can be used as an efficient and reliable alternative to the
	// X-Forwarded-For mechanism which is not always reliable and not even always
	// usable. See also "tcp-request connection expect-proxy" for a finer-grained
	// setting of which client is allowed to use the protocol.
	// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
	KeepSourceIP = EngressKey + "/" + "keep-source-ip"

	// Annotations applied to resources offshoot from an ingress
	OriginAPISchema = EngressKey + "/" + "origin-api-schema" // APISchema = {APIGroup}/{APIVersion}
	OriginName      = EngressKey + "/" + "origin-name"
)

func (r Ingress) OffshootName() string {
	return VoyagerPrefix + r.Name
}

func (r Ingress) APISchema() string {
	if v, ok := getString(r.Annotations, APISchema); ok {
		return v
	}
	return APISchemaEngress
}

func (r Ingress) StickySession() bool {
	return getBool(r.Annotations, StickySession)
}

func (r Ingress) Stats() bool {
	return getBool(r.Annotations, StatsOn)
}

func (r Ingress) StatsSecretName() string {
	v, _ := getString(r.Annotations, StatsSecret)
	return v
}

func (r Ingress) StatsPort() int {
	if v, ok := getInt(r.Annotations, StatsPort); ok {
		return v
	}
	return DefaultStatsPort
}

func (r Ingress) StatsServiceName() string {
	if v, ok := getString(r.Annotations, StatsServiceName); ok {
		return v
	}
	return r.Name + "-stats"
}

func (r Ingress) LBType() string {
	if v, ok := getString(r.Annotations, LBType); ok {
		return v
	}
	return LBTypeLoadBalancer
}

func (r Ingress) Replicas() int32 {
	if v, ok := getInt(r.Annotations, Replicas); ok {
		return int32(v)
	}
	return 1
}

func (r Ingress) NodeSelector() map[string]string {
	if v, ok := getMap(r.Annotations, NodeSelector); ok {
		return v
	}
	v, _ := getString(r.Annotations, EngressKey+"/"+"daemon.nodeSelector")
	return parseNodeSelector(v)
}

func (r Ingress) Persist() string {
	if v, ok := r.Annotations[EngressKey+"/"+"ip"]; ok {
		return v
	}
	v, _ := r.Annotations[LoadBalancerPersist]
	return v
}

func (r Ingress) ServiceAnnotations(provider string) (map[string]string, bool) {
	ans, ok := getMap(r.Annotations, ServiceAnnotations)
	if ok {
		filteredMap := make(map[string]string)
		for k, v := range ans {
			if !strings.HasPrefix(strings.TrimSpace(k), EngressKey+"/") {
				filteredMap[k] = v
			}
		}
		if r.LBType() == LBTypeLoadBalancer && r.KeepSourceIP() {
			switch provider {
			case "aws":
				// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
				filteredMap["service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"] = "*"
			case "gce", "gke", "azure":
				// ref: https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-typeloadbalancer
				filteredMap["service.beta.kubernetes.io/external-traffic"] = "OnlyLocal"
			}
		}
		return filteredMap, true
	}
	return ans, ok
}

func (r Ingress) PodsAnnotations() (map[string]string, bool) {
	ans, ok := getMap(r.Annotations, PodAnnotations)
	if ok {
		filteredMap := make(map[string]string)
		for k, v := range ans {
			if !strings.HasPrefix(strings.TrimSpace(k), EngressKey+"/") {
				filteredMap[k] = v
			}
		}
		return filteredMap, true
	}
	return ans, ok
}

func (r Ingress) KeepSourceIP() bool {
	return getBool(r.Annotations, KeepSourceIP)
}

// ref: https://github.com/kubernetes/kubernetes/blob/078238a461a0872a8eacb887fbb3d0085714604c/staging/src/k8s.io/apiserver/pkg/apis/example/v1/types.go#L134
func parseNodeSelector(labels string) map[string]string {
	selectorMap := make(map[string]string)
	for _, label := range strings.Split(labels, ",") {
		label = strings.TrimSpace(label)
		if len(label) > 0 && strings.Contains(label, "=") {
			data := strings.SplitN(label, "=", 2)
			if len(data) >= 2 {
				if len(data[0]) > 0 && len(data[1]) > 0 {
					selectorMap[data[0]] = data[1]
				}
			}
		}
	}
	return selectorMap
}

func getBool(m map[string]string, key string) bool {
	if m == nil {
		return false
	}
	v, _ := strconv.ParseBool(m[key])
	return v
}

func getInt(m map[string]string, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	s, ok := m[key]
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	return v, err == nil
}

func getString(m map[string]string, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func getList(m map[string]string, key string) ([]string, bool) {
	if m == nil {
		return []string{}, false
	}
	s, ok := m[key]
	if !ok {
		return []string{}, false
	}
	v := make([]string, 0)
	err := json.Unmarshal([]byte(s), &v)
	return v, err == nil
}

func getMap(m map[string]string, key string) (map[string]string, bool) {
	if m == nil {
		return map[string]string{}, false
	}
	s, ok := m[key]
	if !ok {
		return map[string]string{}, false
	}
	v := make(map[string]string)
	err := json.Unmarshal([]byte(s), &v)
	return v, err == nil
}
