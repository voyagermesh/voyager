package api

import (
	"fmt"

	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	/*
		// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3
			resolvers mydns
			  nameserver dns1 10.0.0.1:53
			  nameserver dns2 10.0.0.2:53
			  resolve_retries       3
			  timeout retry         1s
			  hold other           30s
			  hold refused         30s
			  hold nx              30s
			  hold timeout         30s
			  hold valid           10s
	*/
	UseDNSResolver         = EngressKey + "/" + "use-dns-resolver"          // Bool
	DNSResolverNameservers = EngressKey + "/" + "dns-resolver-nameservers"  // ["", ""]
	DNSResolverCheckHealth = EngressKey + "/" + "dns-resolver-check-health" // bool
	DNSResolverRetries     = EngressKey + "/" + "dns-resolver-retries"      // int
	DNSResolverTimeout     = EngressKey + "/" + "dns-resolver-timeout"      // {"event":"time","event":"time"}
	DNSResolverHold        = EngressKey + "/" + "dns-resolver-hold"         // {"status":"period","status":"period"}
)

type DNSResolver struct {
	Name        string
	NameServer  []string          `json:"nameserver"`
	CheckHealth bool              `json:"checkHealth"`
	Retries     int               `json:"retries"`
	Timeout     map[string]string `json:"timeout"`
	Hold        map[string]string `json:"hold"`
}

func DNSResolverForService(svc apiv1.Service) (useDNSResolver bool, resolver *DNSResolver, err error) {
	if svc.Spec.Type != apiv1.ServiceTypeExternalName {
		return false, nil, fmt.Errorf("Service %s@%s is expected to be of type ServiceTypeExternalName, actual type %s", svc.Name, svc.Namespace, svc.Spec.Type)
	}
	useDNSResolver, err = getBool(svc.Annotations, UseDNSResolver)
	if err != nil || !useDNSResolver {
		return
	}

	resolver = &DNSResolver{Name: svc.Spec.ExternalName}
	resolver.NameServer, err = getList(svc.Annotations, DNSResolverNameservers)
	if err != nil {
		return
	}
	if ch, e2 := getBool(svc.Annotations, DNSResolverCheckHealth); e2 == nil {
		resolver.CheckHealth = ch
	} else {
		resolver.CheckHealth = len(resolver.NameServer) > 0
	}
	resolver.Retries, err = getInt(svc.Annotations, DNSResolverRetries)
	if err != nil {
		return
	}
	resolver.Hold, err = getMap(svc.Annotations, DNSResolverHold)
	if err != nil {
		return
	}
	resolver.Timeout, err = getMap(svc.Annotations, DNSResolverTimeout)
	return
}
