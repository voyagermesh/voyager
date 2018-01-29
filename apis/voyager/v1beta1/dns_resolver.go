package v1beta1

import (
	"fmt"

	"github.com/appscode/kutil"
	"github.com/appscode/kutil/meta"
	core "k8s.io/api/core/v1"
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

func DNSResolverForService(svc core.Service) (useDNSResolver bool, resolver *DNSResolver, err error) {
	if svc.Spec.Type != core.ServiceTypeExternalName {
		return false, nil, fmt.Errorf("service %s/%s is expected to be of type ServiceTypeExternalName, actual type %s", svc.Namespace, svc.Name, svc.Spec.Type)
	}

	// getBool returns an error if the value is empty string, or the key is not present
	// So, we ignored error.
	useDNSResolver, _ = meta.GetBool(svc.Annotations, UseDNSResolver)
	if !useDNSResolver {
		return
	}

	resolver = &DNSResolver{Name: svc.Spec.ExternalName}
	resolver.NameServer, err = meta.GetList(svc.Annotations, DNSResolverNameservers)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	if ch, e2 := meta.GetBool(svc.Annotations, DNSResolverCheckHealth); e2 == nil {
		resolver.CheckHealth = ch
	} else {
		resolver.CheckHealth = len(resolver.NameServer) > 0
	}
	resolver.Retries, err = meta.GetInt(svc.Annotations, DNSResolverRetries)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	resolver.Hold, err = meta.GetMap(svc.Annotations, DNSResolverHold)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	resolver.Timeout, err = meta.GetMap(svc.Annotations, DNSResolverTimeout)
	if err == kutil.ErrNotFound {
		err = nil
	}
	return
}
