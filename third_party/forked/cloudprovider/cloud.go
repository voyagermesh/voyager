/*
Copyright 2014 The Kubernetes Authors.

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

package cloudprovider

import (
	"errors"
	"strings"
apiv1 "k8s.io/client-go/pkg/api/v1"
)

// Interface is an abstract, pluggable interface for cloud providers.
type Interface interface {
	// Firewall returns a firewall interface. Also returns true if the interface is supported, false otherwise.
	Firewall() (Firewall, bool)
	// ProviderName returns the cloud provider ID.
	ProviderName() string
}

// TODO(#6812): Use a shorter name that's less likely to be longer than cloud
// providers' name length limits.
func GetLoadBalancerName(service *apiv1.Service) string {
	//GCE requires that the name of a load balancer starts with a lower case letter.
	ret := "a" + string(service.UID)
	ret = strings.Replace(ret, "-", "", -1)
	//AWS requires that the name of a load balancer is shorter than 32 bytes.
	if len(ret) > 32 {
		ret = ret[:32]
	}
	return ret
}

// Firewall is an abstract, pluggable interface for firewalls.
type Firewall interface {
	// EnsureFirewall creates and/or update firewall rules.
	// Implementations must treat the *apiv1.Service parameter as read-only and not modify it.
	EnsureFirewall(service *apiv1.Service, hostname string) error

	// EnsureFirewallDeleted deletes the specified firewall if it
	// exists, returning nil if the firewall specified either didn't exist or
	// was successfully deleted.
	// This construction is useful because many cloud providers' firewall
	// have multiple underlying components, meaning a Get could say that the firewall
	// doesn't exist even if some part of it is still laying around.
	// Implementations must treat the *apiv1.Service parameter as read-only and not modify it.
	EnsureFirewallDeleted(service *apiv1.Service) error
}

var InstanceNotFound = errors.New("instance not found")
