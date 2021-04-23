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
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kutil "kmodules.xyz/client-go"
	"kmodules.xyz/client-go/meta"
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

var (
	DNSResolverHoldKeys    = sets.NewString("nx", "other", "refused", "timeout", "valid", "obsolete")
	DNSResolverTimeoutKeys = sets.NewString("resolve", "retry")
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
		return false, nil, errors.Errorf("service %s/%s is expected to be of type ServiceTypeExternalName, actual type %s", svc.Namespace, svc.Name, svc.Spec.Type)
	}

	// getBool returns an error if the value is empty string, or the key is not present
	// So, we ignored error.
	useDNSResolver, _ = meta.GetBoolValue(svc.Annotations, UseDNSResolver)
	if !useDNSResolver {
		return
	}

	resolver = &DNSResolver{Name: svc.Spec.ExternalName}
	resolver.NameServer, err = meta.GetListValue(svc.Annotations, DNSResolverNameservers)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	if ch, e2 := meta.GetBoolValue(svc.Annotations, DNSResolverCheckHealth); e2 == nil {
		resolver.CheckHealth = ch
	} else {
		resolver.CheckHealth = len(resolver.NameServer) > 0
	}
	resolver.Retries, err = meta.GetIntValue(svc.Annotations, DNSResolverRetries)
	if err != nil && err != kutil.ErrNotFound {
		return
	}

	resolver.Hold, err = meta.GetMapValue(svc.Annotations, DNSResolverHold)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	if err = checkMapKeys(resolver.Hold, DNSResolverHoldKeys); err != nil {
		err = errors.Errorf("invalid value for annotaion %s. Reason: %s", DNSResolverHold, err)
		return
	}

	resolver.Timeout, err = meta.GetMapValue(svc.Annotations, DNSResolverTimeout)
	if err != nil && err != kutil.ErrNotFound {
		return
	}
	if err = checkMapKeys(resolver.Timeout, DNSResolverTimeoutKeys); err != nil {
		err = errors.Errorf("invalid value for annotaion %s. Reason: %s", DNSResolverTimeout, err)
		return
	}

	return
}
