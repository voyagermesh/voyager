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

package fake

import (
	"errors"
	"net"
	"regexp"
	"sync"

	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const ProviderName = "fake"

// FakeBalancer is a fake storage of balancer information
type FakeBalancer struct {
	Name           string
	Region         string
	LoadBalancerIP string
	Ports          []apiv1.ServicePort
	Hosts          []string
}

type FakeUpdateBalancerCall struct {
	Service *apiv1.Service
	Hosts   []string
}

// FakeCloud is a test-double implementation of Interface, LoadBalancer, Instances, and Routes. It is useful for testing.
type FakeCloud struct {
	Exists        bool
	Err           error
	Calls         []string
	Addresses     []apiv1.NodeAddress
	ExtID         map[types.NodeName]string
	InstanceTypes map[types.NodeName]string
	Machines      []types.NodeName
	NodeResources *apiv1.NodeResources
	ClusterList   []string
	MasterName    string
	ExternalIP    net.IP
	Balancers     map[string]FakeBalancer
	UpdateCalls   []FakeUpdateBalancerCall
	RouteMap      map[string]*FakeRoute
	Lock          sync.Mutex
}

type FakeRoute struct {
	ClusterName string
}

func (f *FakeCloud) addCall(desc string) {
	f.Calls = append(f.Calls, desc)
}

// ClearCalls clears internal record of method calls to this FakeCloud.
func (f *FakeCloud) ClearCalls() {
	f.Calls = []string{}
}

// ProviderName returns the cloud provider ID.
func (f *FakeCloud) ProviderName() string {
	return ProviderName
}

// Firewall returns a nil Firewall.
func (f *FakeCloud) Firewall() (cloudprovider.Firewall, bool) {
	return nil, false
}

// GetLoadBalancer is a stub implementation of LoadBalancer.GetLoadBalancer.
func (f *FakeCloud) GetLoadBalancer(clusterName string, service *apiv1.Service) (*apiv1.LoadBalancerStatus, bool, error) {
	status := &apiv1.LoadBalancerStatus{}
	status.Ingress = []apiv1.LoadBalancerIngress{{IP: f.ExternalIP.String()}}

	return status, f.Exists, f.Err
}

func (f *FakeCloud) AddSSHKeyToAllInstances(user string, keyData []byte) error {
	return errors.New("unimplemented")
}

// Implementation of Instances.CurrentNodeName
func (f *FakeCloud) CurrentNodeName(hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

// NodeAddresses is a test-spy implementation of Instances.NodeAddresses.
// It adds an entry "node-addresses" into the internal method call record.
func (f *FakeCloud) NodeAddresses(instance types.NodeName) ([]apiv1.NodeAddress, error) {
	f.addCall("node-addresses")
	return f.Addresses, f.Err
}

// ExternalID is a test-spy implementation of Instances.ExternalID.
// It adds an entry "external-id" into the internal method call record.
// It returns an external id to the mapped instance name, if not found, it will return "ext-{instance}"
func (f *FakeCloud) ExternalID(nodeName types.NodeName) (string, error) {
	f.addCall("external-id")
	return f.ExtID[nodeName], f.Err
}

// InstanceID returns the cloud provider ID of the node with the specified Name.
func (f *FakeCloud) InstanceID(nodeName types.NodeName) (string, error) {
	f.addCall("instance-id")
	return f.ExtID[nodeName], nil
}

// InstanceType returns the type of the specified instance.
func (f *FakeCloud) InstanceType(instance types.NodeName) (string, error) {
	f.addCall("instance-type")
	return f.InstanceTypes[instance], nil
}

// List is a test-spy implementation of Instances.List.
// It adds an entry "list" into the internal method call record.
func (f *FakeCloud) List(filter string) ([]types.NodeName, error) {
	f.addCall("list")
	result := []types.NodeName{}
	for _, machine := range f.Machines {
		if match, _ := regexp.MatchString(filter, string(machine)); match {
			result = append(result, machine)
		}
	}
	return result, f.Err
}
