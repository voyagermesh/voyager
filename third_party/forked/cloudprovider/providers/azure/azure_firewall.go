package azure

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"voyagermesh.dev/voyager/third_party/forked/cloudprovider"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

// EnsureFirewall creates and/or update firewall rules.
func (az *Cloud) EnsureFirewall(ctx context.Context, service *apiv1.Service, hostnames []string) error {
	serviceName := getServiceName(service)
	klog.V(2).Infof("ensure(%s): START EnsureFirewall", serviceName)
	hostname := hostnames[0]

	machine, exists, err := az.getVirtualMachine(ctx, types.NodeName(hostname))
	if err != nil {
		return err
	} else if !exists {
		return cloudprovider.InstanceNotFound
	}

	var nicName string
	for _, nic := range *machine.NetworkProfile.NetworkInterfaces {
		if *nic.Primary {
			nicName = (*nic.ID)[strings.LastIndex(*nic.ID, "/")+1:]
			break
		}
	}

	nic, err := az.InterfacesClient.Get(ctx, az.ResourceGroup, nicName, "")
	exists, realErr := checkResourceExistsFromError(err)
	if realErr != nil {
		return realErr
	}
	if !exists {
		return errors.Errorf("Failed to detect internal ip for host %v", hostname)
	}
	internlIP := *(*nic.IPConfigurations)[0].PrivateIPAddress

	sg, err := az.SecurityGroupsClient.Get(ctx, az.ResourceGroup, az.SecurityGroupName, "")
	if err != nil {
		return err
	}

	sg, sgNeedsUpdate, err := az.reconcileFirewall(sg, service, internlIP)
	if err != nil {
		return err
	}
	if sgNeedsUpdate {
		klog.V(3).Infof("ensure(%s): sg(%s) - updating", serviceName, *sg.Name)
		_, err := az.SecurityGroupsClient.CreateOrUpdate(ctx, az.ResourceGroup, *sg.Name, sg)
		if err != nil {
			return err
		}
	}

	klog.V(2).Infof("ensure(%s): FINISH EnsureFirewall", service.Name)
	return nil
}

// EnsureFirewallDeleted deletes the specified firewall rules if those
// exist, returning nil if the firewall rules specified either didn't exist or
// was successfully deleted.
func (az *Cloud) EnsureFirewallDeleted(ctx context.Context, service *apiv1.Service) error {
	serviceName := getServiceName(service)

	klog.V(2).Infof("delete(%s): START EnsureFirewallDeleted", serviceName)

	// reconcile logic is capable of fully reconcile, so we can use this to delete
	service.Spec.Ports = []apiv1.ServicePort{}

	sg, existsSg, err := az.getSecurityGroup(ctx)
	if err != nil {
		return err
	}
	if existsSg {
		// hack: We expect no new additions, so we can pass * as the destination address.
		reconciledSg, sgNeedsUpdate, reconcileErr := az.reconcileFirewall(sg, service, "*")
		if reconcileErr != nil {
			return reconcileErr
		}
		if sgNeedsUpdate {
			klog.V(3).Infof("delete(%s): sg(%s) - updating", serviceName, az.SecurityGroupName)
			_, err := az.SecurityGroupsClient.CreateOrUpdate(ctx, az.ResourceGroup, *reconciledSg.Name, reconciledSg)
			if err != nil {
				return err
			}
		}
	}

	klog.V(2).Infof("delete(%s): FINISH EnsureFirewallDeleted", serviceName)
	return nil
}

// This reconciles the Network Security Group similar to how the LB is reconciled.
// This entails adding required, missing SecurityRules and removing stale rules.
func (az *Cloud) reconcileFirewall(sg network.SecurityGroup, service *apiv1.Service, destAddr string) (network.SecurityGroup, bool, error) {
	serviceName := getServiceName(service)
	wantLb := len(service.Spec.Ports) > 0
	expectedSecurityRules := make([]network.SecurityRule, len(service.Spec.Ports))
	for i, port := range service.Spec.Ports {
		securityRuleName := getFwRuleName(service, port)
		_, securityProto, _, err := getProtocolsFromKubernetesProtocol(port.Protocol)
		if err != nil {
			return sg, false, err
		}

		expectedSecurityRules[i] = network.SecurityRule{
			Name: to.StringPtr(securityRuleName),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Protocol:                 securityProto,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(strconv.Itoa(int(port.NodePort))),
				SourceAddressPrefix:      to.StringPtr("Internet"),
				DestinationAddressPrefix: to.StringPtr(destAddr),
				Access:                   network.SecurityRuleAccessAllow,
				Direction:                network.SecurityRuleDirectionInbound,
			},
		}
	}

	// update security rules
	dirtySg := false
	var updatedRules []network.SecurityRule
	if sg.SecurityRules != nil {
		updatedRules = *sg.SecurityRules
	}
	// update security rules: remove unwanted
	for i := len(updatedRules) - 1; i >= 0; i-- {
		existingRule := updatedRules[i]
		if serviceOwnsRule(service, *existingRule.Name) {
			klog.V(10).Infof("reconcile(%s)(%t): sg rule(%s) - considering evicting", serviceName, wantLb, *existingRule.Name)
			keepRule := false
			if findSecurityRule(expectedSecurityRules, existingRule) {
				klog.V(10).Infof("reconcile(%s)(%t): sg rule(%s) - keeping", serviceName, wantLb, *existingRule.Name)
				keepRule = true
			}
			if !keepRule {
				klog.V(10).Infof("reconcile(%s)(%t): sg rule(%s) - dropping", serviceName, wantLb, *existingRule.Name)
				updatedRules = append(updatedRules[:i], updatedRules[i+1:]...)
				dirtySg = true
			}
		}
	}
	// update security rules: add needed
	for _, expectedRule := range expectedSecurityRules {
		foundRule := false
		if findSecurityRule(updatedRules, expectedRule) {
			klog.V(10).Infof("reconcile(%s)(%t): sg rule(%s) - already exists", serviceName, wantLb, *expectedRule.Name)
			foundRule = true
		}
		if !foundRule {
			klog.V(10).Infof("reconcile(%s)(%t): sg rule(%s) - adding", serviceName, wantLb, *expectedRule.Name)

			nextAvailablePriority, err := getNextAvailablePriority(updatedRules)
			if err != nil {
				return sg, false, err
			}

			expectedRule.Priority = to.Int32Ptr(nextAvailablePriority)
			updatedRules = append(updatedRules, expectedRule)
			dirtySg = true
		}
	}
	if dirtySg {
		sg.SecurityRules = &updatedRules
	}
	return sg, dirtySg, nil
}

func getFwRuleName(service *apiv1.Service, port apiv1.ServicePort) string {
	return fmt.Sprintf("%s-%s-fw-%d", getRulePrefix(service), port.Protocol, port.TargetPort.IntValue())
}
