/*
Copyright 2016 The Kubernetes Authors.

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

package azure

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	apiv1 "k8s.io/api/core/v1"
)

const (
	loadBalancerMinimumPriority = 500
	loadBalancerMaximumPriority = 4096
)

// returns the equivalent LoadBalancerRule, SecurityRule and LoadBalancerProbe
// protocol types for the given Kubernetes protocol type.
func getProtocolsFromKubernetesProtocol(protocol apiv1.Protocol) (network.TransportProtocol, network.SecurityRuleProtocol, network.ProbeProtocol, error) {
	switch protocol {
	case apiv1.ProtocolTCP:
		return network.TransportProtocolTCP, network.SecurityRuleProtocolTCP, network.ProbeProtocolTCP, nil
	default:
		return "", "", "", fmt.Errorf("Only TCP is supported for Azure LoadBalancers")
	}
}

// This returns a human-readable version of the Service used to tag some resources.
// This is only used for human-readable convenience, and not to filter.
func getServiceName(service *apiv1.Service) string {
	return fmt.Sprintf("%s/%s", service.Namespace, service.Name)
}

// This returns a prefix for loadbalancer/security rules.
func getRulePrefix(service *apiv1.Service) string {
	return cloudprovider.GetSecurityGroupName(service)
}

func serviceOwnsRule(service *apiv1.Service, rule string) bool {
	prefix := getRulePrefix(service)
	return strings.HasPrefix(strings.ToUpper(rule), strings.ToUpper(prefix))
}

// This returns the next available rule priority level for a given set of security rules.
func getNextAvailablePriority(rules []network.SecurityRule) (int32, error) {
	var smallest int32 = loadBalancerMinimumPriority
	var spread int32 = 1

outer:
	for smallest < loadBalancerMaximumPriority {
		for _, rule := range rules {
			if *rule.Priority == smallest {
				smallest += spread
				continue outer
			}
		}
		// no one else had it
		return smallest, nil
	}

	return -1, fmt.Errorf("SecurityGroup priorities are exhausted")
}
