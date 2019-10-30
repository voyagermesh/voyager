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
	"io"
	"io/ioutil"
	"time"

	"github.com/appscode/voyager/third_party/forked/cloudprovider"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"sigs.k8s.io/yaml"
)

// CloudProviderName is the value used for the --cloud-provider flag
const CloudProviderName = "azure"

// Config holds the configuration parsed from the --cloud-config flag
type Config struct {
	Cloud                      string `json:"cloud" yaml:"cloud"`
	TenantID                   string `json:"tenantId" yaml:"tenantId"`
	SubscriptionID             string `json:"subscriptionId" yaml:"subscriptionId"`
	ResourceGroup              string `json:"resourceGroup" yaml:"resourceGroup"`
	Location                   string `json:"location" yaml:"location"`
	VnetName                   string `json:"vnetName" yaml:"vnetName"`
	SubnetName                 string `json:"subnetName" yaml:"subnetName"`
	SecurityGroupName          string `json:"securityGroupName" yaml:"securityGroupName"`
	RouteTableName             string `json:"routeTableName" yaml:"routeTableName"`
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName" yaml:"primaryAvailabilitySetName"`

	AADClientID     string `json:"aadClientId" yaml:"aadClientId"`
	AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	AADTenantID     string `json:"aadTenantId" yaml:"aadTenantId"`
}

// Cloud holds the config and clients
type Cloud struct {
	Config
	Environment           azure.Environment
	InterfacesClient      network.InterfacesClient
	SecurityGroupsClient  network.SecurityGroupsClient
	VirtualMachinesClient compute.VirtualMachinesClient
}

func init() {
	cloudprovider.RegisterCloudProvider(CloudProviderName, NewCloud)
}

// NewCloud returns a Cloud with initialized clients
func NewCloud(configReader io.Reader) (cloudprovider.Interface, error) {
	var az Cloud

	configContents, err := ioutil.ReadAll(configReader)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(configContents, &az)
	if err != nil {
		return nil, err
	}

	if az.Cloud == "" {
		az.Environment = azure.PublicCloud
	} else {
		az.Environment, err = azure.EnvironmentFromName(az.Cloud)
		if err != nil {
			return nil, err
		}
	}

	oauthConfig, err := adal.NewOAuthConfig(az.Environment.ActiveDirectoryEndpoint, az.TenantID)
	if err != nil {
		return nil, err
	}

	servicePrincipalToken, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		az.AADClientID,
		az.AADClientSecret,
		az.Environment.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	az.InterfacesClient = network.NewInterfacesClient(az.SubscriptionID)
	az.InterfacesClient.BaseURI = az.Environment.ResourceManagerEndpoint
	az.InterfacesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

	az.VirtualMachinesClient = compute.NewVirtualMachinesClient(az.SubscriptionID)
	az.VirtualMachinesClient.BaseURI = az.Environment.ResourceManagerEndpoint
	az.VirtualMachinesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	az.VirtualMachinesClient.PollingDelay = 5 * time.Second

	az.SecurityGroupsClient = network.NewSecurityGroupsClient(az.SubscriptionID)
	az.SecurityGroupsClient.BaseURI = az.Environment.ResourceManagerEndpoint
	az.SecurityGroupsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

	return &az, nil
}

// Firewall returns a firewall interface. Also returns true if the interface is supported, false otherwise.
func (az *Cloud) Firewall() (cloudprovider.Firewall, bool) {
	return az, false
}

// ProviderName returns the cloud provider ID.
func (az *Cloud) ProviderName() string {
	return CloudProviderName
}
