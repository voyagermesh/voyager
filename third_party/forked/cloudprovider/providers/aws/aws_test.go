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

package aws

import (
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

const TestClusterId = "clusterid.test"
const TestClusterName = "testCluster"

func TestReadAWSCloudConfig(t *testing.T) {
	tests := []struct {
		name string

		reader io.Reader
		aws    Services

		expectError bool
		zone        string
	}{
		{
			"No config reader",
			nil, nil,
			true, "",
		},
		{
			"Empty config, no metadata",
			strings.NewReader(""), nil,
			true, "",
		},
		{
			"No zone in config, no metadata",
			strings.NewReader("[global]\n"), nil,
			true, "",
		},
		{
			"Zone in config, no metadata",
			strings.NewReader("[global]\nzone = eu-west-1a"), nil,
			false, "eu-west-1a",
		},
		{
			"No zone in config, metadata does not have zone",
			strings.NewReader("[global]\n"), NewFakeAWSServices().withAz(""),
			true, "",
		},
		{
			"No zone in config, metadata has zone",
			strings.NewReader("[global]\n"), NewFakeAWSServices(),
			false, "us-east-1a",
		},
		{
			"Zone in config should take precedence over metadata",
			strings.NewReader("[global]\nzone = eu-west-1a"), NewFakeAWSServices(),
			false, "eu-west-1a",
		},
	}

	for _, test := range tests {
		t.Logf("Running test case %s", test.name)
		var metadata EC2Metadata
		if test.aws != nil {
			metadata, _ = test.aws.Metadata()
		}
		cfg, err := readAWSCloudConfig(test.reader, metadata)
		if test.expectError {
			if err == nil {
				t.Errorf("Should error for case %s (cfg=%v)", test.name, cfg)
			}
		} else {
			if err != nil {
				t.Errorf("Should succeed for case: %s", test.name)
			}
			if cfg.Global.Zone != test.zone {
				t.Errorf("Incorrect zone value (%s vs %s) for case: %s",
					cfg.Global.Zone, test.zone, test.name)
			}
		}
	}
}

type FakeAWSServices struct {
	region                  string
	instances               []*ec2.Instance
	selfInstance            *ec2.Instance
	networkInterfacesMacs   []string
	networkInterfacesVpcIDs []string

	ec2      *FakeEC2
	elb      *FakeELB
	asg      *FakeASG
	metadata *FakeMetadata
}

func NewFakeAWSServices() *FakeAWSServices {
	s := &FakeAWSServices{}
	s.region = "us-east-1"
	s.ec2 = &FakeEC2{aws: s}
	s.elb = &FakeELB{aws: s}
	s.asg = &FakeASG{aws: s}
	s.metadata = &FakeMetadata{aws: s}

	s.networkInterfacesMacs = []string{"aa:bb:cc:dd:ee:00", "aa:bb:cc:dd:ee:01"}
	s.networkInterfacesVpcIDs = []string{"vpc-mac0", "vpc-mac1"}

	selfInstance := &ec2.Instance{}
	selfInstance.InstanceId = aws.String("i-self")
	selfInstance.Placement = &ec2.Placement{
		AvailabilityZone: aws.String("us-east-1a"),
	}
	selfInstance.PrivateDnsName = aws.String("ip-172-20-0-100.ec2.internal")
	selfInstance.PrivateIpAddress = aws.String("192.168.0.1")
	selfInstance.PublicIpAddress = aws.String("1.2.3.4")
	s.selfInstance = selfInstance
	s.instances = []*ec2.Instance{selfInstance}

	var tag ec2.Tag
	tag.Key = aws.String(TagNameKubernetesCluster)
	tag.Value = aws.String(TestClusterId)
	selfInstance.Tags = []*ec2.Tag{&tag}

	return s
}

func (s *FakeAWSServices) withAz(az string) *FakeAWSServices {
	if s.selfInstance.Placement == nil {
		s.selfInstance.Placement = &ec2.Placement{}
	}
	s.selfInstance.Placement.AvailabilityZone = aws.String(az)
	return s
}

func (s *FakeAWSServices) Compute(region string) (EC2, error) {
	return s.ec2, nil
}

func (s *FakeAWSServices) LoadBalancing(region string) (ELB, error) {
	return s.elb, nil
}

func (s *FakeAWSServices) Metadata() (EC2Metadata, error) {
	return s.metadata, nil
}

func TestFilterTags(t *testing.T) {
	awsServices := NewFakeAWSServices()
	c, err := newAWSCloud(strings.NewReader("[global]"), awsServices)
	if err != nil {
		t.Errorf("Error building aws cloud: %v", err)
		return
	}

	if len(c.filterTags) != 1 {
		t.Errorf("unexpected filter tags: %v", c.filterTags)
		return
	}

	if c.filterTags[TagNameKubernetesCluster] != TestClusterId {
		t.Errorf("unexpected filter tags: %v", c.filterTags)
	}
}

func TestNewAWSCloud(t *testing.T) {
	tests := []struct {
		name string

		reader      io.Reader
		awsServices Services

		expectError bool
		region      string
	}{
		{
			"No config reader",
			nil, NewFakeAWSServices().withAz(""),
			true, "",
		},
		{
			"Config specified invalid zone",
			strings.NewReader("[global]\nzone = blahonga"), NewFakeAWSServices(),
			true, "",
		},
		{
			"Config specifies valid zone",
			strings.NewReader("[global]\nzone = eu-west-1a"), NewFakeAWSServices(),
			false, "eu-west-1",
		},
		{
			"Gets zone from metadata when not in config",
			strings.NewReader("[global]\n"),
			NewFakeAWSServices(),
			false, "us-east-1",
		},
		{
			"No zone in config or metadata",
			strings.NewReader("[global]\n"),
			NewFakeAWSServices().withAz(""),
			true, "",
		},
	}

	for _, test := range tests {
		t.Logf("Running test case %s", test.name)
		c, err := newAWSCloud(test.reader, test.awsServices)
		if test.expectError {
			if err == nil {
				t.Errorf("Should error for case %s", test.name)
			}
		} else {
			if err != nil {
				t.Errorf("Should succeed for case: %s, got %v", test.name, err)
			} else if c.region != test.region {
				t.Errorf("Incorrect region value (%s vs %s) for case: %s",
					c.region, test.region, test.name)
			}
		}
	}
}

type FakeEC2 struct {
	aws                      *FakeAWSServices
	Subnets                  []*ec2.Subnet
	DescribeSubnetsInput     *ec2.DescribeSubnetsInput
	RouteTables              []*ec2.RouteTable
	DescribeRouteTablesInput *ec2.DescribeRouteTablesInput
	mock.Mock
}

func contains(haystack []*string, needle string) bool {
	for _, s := range haystack {
		// (deliberately panic if s == nil)
		if needle == *s {
			return true
		}
	}
	return false
}

func instanceMatchesFilter(instance *ec2.Instance, filter *ec2.Filter) bool {
	name := *filter.Name
	if name == "private-dns-name" {
		if instance.PrivateDnsName == nil {
			return false
		}
		return contains(filter.Values, *instance.PrivateDnsName)
	}

	if name == "instance-state-name" {
		return contains(filter.Values, *instance.State.Name)
	}

	if strings.HasPrefix(name, "tag:") {
		tagName := name[4:]
		for _, instanceTag := range instance.Tags {
			if aws.StringValue(instanceTag.Key) == tagName && contains(filter.Values, aws.StringValue(instanceTag.Value)) {
				return true
			}
		}
	}
	panic("Unknown filter name: " + name)
}

func (self *FakeEC2) DescribeInstances(request *ec2.DescribeInstancesInput) ([]*ec2.Instance, error) {
	matches := []*ec2.Instance{}
	for _, instance := range self.aws.instances {
		if request.InstanceIds != nil {
			if instance.InstanceId == nil {
				klog.Warning("Instance with no instance id: ", instance)
				continue
			}

			found := false
			for _, instanceID := range request.InstanceIds {
				if *instanceID == *instance.InstanceId {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if request.Filters != nil {
			allMatch := true
			for _, filter := range request.Filters {
				if !instanceMatchesFilter(instance, filter) {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
		}
		matches = append(matches, instance)
	}

	return matches, nil
}

type FakeMetadata struct {
	aws *FakeAWSServices
}

func (self *FakeMetadata) GetMetadata(key string) (string, error) {
	networkInterfacesPrefix := "network/interfaces/macs/"
	i := self.aws.selfInstance
	if key == "placement/availability-zone" {
		az := ""
		if i.Placement != nil {
			az = aws.StringValue(i.Placement.AvailabilityZone)
		}
		return az, nil
	} else if key == "instance-id" {
		return aws.StringValue(i.InstanceId), nil
	} else if key == "local-hostname" {
		return aws.StringValue(i.PrivateDnsName), nil
	} else if key == "local-ipv4" {
		return aws.StringValue(i.PrivateIpAddress), nil
	} else if key == "public-ipv4" {
		return aws.StringValue(i.PublicIpAddress), nil
	} else if strings.HasPrefix(key, networkInterfacesPrefix) {
		if key == networkInterfacesPrefix {
			return strings.Join(self.aws.networkInterfacesMacs, "/\n") + "/\n", nil
		} else {
			keySplit := strings.Split(key, "/")
			macParam := keySplit[3]
			if len(keySplit) == 5 && keySplit[4] == "vpc-id" {
				for i, macElem := range self.aws.networkInterfacesMacs {
					if macParam == macElem {
						return self.aws.networkInterfacesVpcIDs[i], nil
					}
				}
			}
			return "", nil
		}
	} else {
		return "", nil
	}
}

func (ec2 *FakeEC2) AttachVolume(request *ec2.AttachVolumeInput) (resp *ec2.VolumeAttachment, err error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DetachVolume(request *ec2.DetachVolumeInput) (resp *ec2.VolumeAttachment, err error) {
	panic("Not implemented")
}

func (e *FakeEC2) DescribeVolumes(request *ec2.DescribeVolumesInput) ([]*ec2.Volume, error) {
	args := e.Called(request)
	return args.Get(0).([]*ec2.Volume), nil
}

func (ec2 *FakeEC2) CreateVolume(request *ec2.CreateVolumeInput) (resp *ec2.Volume, err error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DeleteVolume(request *ec2.DeleteVolumeInput) (resp *ec2.DeleteVolumeOutput, err error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DescribeSecurityGroups(request *ec2.DescribeSecurityGroupsInput) ([]*ec2.SecurityGroup, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) CreateSecurityGroup(*ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DeleteSecurityGroup(*ec2.DeleteSecurityGroupInput) (*ec2.DeleteSecurityGroupOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) AuthorizeSecurityGroupIngress(*ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) RevokeSecurityGroupIngress(*ec2.RevokeSecurityGroupIngressInput) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DescribeSubnets(request *ec2.DescribeSubnetsInput) ([]*ec2.Subnet, error) {
	ec2.DescribeSubnetsInput = request
	return ec2.Subnets, nil
}

func (ec2 *FakeEC2) CreateTags(*ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeEC2) DescribeRouteTables(request *ec2.DescribeRouteTablesInput) ([]*ec2.RouteTable, error) {
	ec2.DescribeRouteTablesInput = request
	return ec2.RouteTables, nil
}

func (s *FakeEC2) CreateRoute(request *ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error) {
	panic("Not implemented")
}

func (s *FakeEC2) DeleteRoute(request *ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error) {
	panic("Not implemented")
}

func (s *FakeEC2) ModifyInstanceAttribute(request *ec2.ModifyInstanceAttributeInput) (*ec2.ModifyInstanceAttributeOutput, error) {
	panic("Not implemented")
}

type FakeELB struct {
	aws *FakeAWSServices
	mock.Mock
}

func (ec2 *FakeELB) CreateLoadBalancer(*elb.CreateLoadBalancerInput) (*elb.CreateLoadBalancerOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) DeleteLoadBalancer(input *elb.DeleteLoadBalancerInput) (*elb.DeleteLoadBalancerOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) DescribeLoadBalancers(input *elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error) {
	args := ec2.Called(input)
	return args.Get(0).(*elb.DescribeLoadBalancersOutput), nil
}
func (ec2 *FakeELB) RegisterInstancesWithLoadBalancer(*elb.RegisterInstancesWithLoadBalancerInput) (*elb.RegisterInstancesWithLoadBalancerOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) DeregisterInstancesFromLoadBalancer(*elb.DeregisterInstancesFromLoadBalancerInput) (*elb.DeregisterInstancesFromLoadBalancerOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) DetachLoadBalancerFromSubnets(*elb.DetachLoadBalancerFromSubnetsInput) (*elb.DetachLoadBalancerFromSubnetsOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) AttachLoadBalancerToSubnets(*elb.AttachLoadBalancerToSubnetsInput) (*elb.AttachLoadBalancerToSubnetsOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) CreateLoadBalancerListeners(*elb.CreateLoadBalancerListenersInput) (*elb.CreateLoadBalancerListenersOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) DeleteLoadBalancerListeners(*elb.DeleteLoadBalancerListenersInput) (*elb.DeleteLoadBalancerListenersOutput, error) {
	panic("Not implemented")
}

func (ec2 *FakeELB) ApplySecurityGroupsToLoadBalancer(*elb.ApplySecurityGroupsToLoadBalancerInput) (*elb.ApplySecurityGroupsToLoadBalancerOutput, error) {
	panic("Not implemented")
}

func (elb *FakeELB) ConfigureHealthCheck(*elb.ConfigureHealthCheckInput) (*elb.ConfigureHealthCheckOutput, error) {
	panic("Not implemented")
}

func (elb *FakeELB) CreateLoadBalancerPolicy(*elb.CreateLoadBalancerPolicyInput) (*elb.CreateLoadBalancerPolicyOutput, error) {
	panic("Not implemented")
}

func (elb *FakeELB) SetLoadBalancerPoliciesForBackendServer(*elb.SetLoadBalancerPoliciesForBackendServerInput) (*elb.SetLoadBalancerPoliciesForBackendServerOutput, error) {
	panic("Not implemented")
}

func (elb *FakeELB) DescribeLoadBalancerAttributes(*elb.DescribeLoadBalancerAttributesInput) (*elb.DescribeLoadBalancerAttributesOutput, error) {
	panic("Not implemented")
}

func (elb *FakeELB) ModifyLoadBalancerAttributes(*elb.ModifyLoadBalancerAttributesInput) (*elb.ModifyLoadBalancerAttributesOutput, error) {
	panic("Not implemented")
}

type FakeASG struct {
	aws *FakeAWSServices
}

func (a *FakeASG) UpdateAutoScalingGroup(*autoscaling.UpdateAutoScalingGroupInput) (*autoscaling.UpdateAutoScalingGroupOutput, error) {
	panic("Not implemented")
}

func (a *FakeASG) DescribeAutoScalingGroups(*autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	panic("Not implemented")
}

func TestIpPermissionExistsHandlesMultipleGroupIds(t *testing.T) {
	oldIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("firstGroupId")},
			{GroupId: aws.String("secondGroupId")},
			{GroupId: aws.String("thirdGroupId")},
		},
	}

	existingIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("secondGroupId")},
		},
	}

	newIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("fourthGroupId")},
		},
	}

	equals := ipPermissionExists(&existingIpPermission, &oldIpPermission, false)
	if !equals {
		t.Errorf("Should have been considered equal since first is in the second array of groups")
	}

	equals = ipPermissionExists(&newIpPermission, &oldIpPermission, false)
	if equals {
		t.Errorf("Should have not been considered equal since first is not in the second array of groups")
	}
}

func TestIpPermissionExistsHandlesRangeSubsets(t *testing.T) {
	// Two existing scenarios we'll test against
	emptyIpPermission := ec2.IpPermission{}

	oldIpPermission := ec2.IpPermission{
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("10.0.0.0/8")},
			{CidrIp: aws.String("192.168.1.0/24")},
		},
	}

	// Two already existing ranges and a new one
	existingIpPermission := ec2.IpPermission{
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("10.0.0.0/8")},
		},
	}
	existingIpPermission2 := ec2.IpPermission{
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("192.168.1.0/24")},
		},
	}

	newIpPermission := ec2.IpPermission{
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("172.16.0.0/16")},
		},
	}

	exists := ipPermissionExists(&emptyIpPermission, &emptyIpPermission, false)
	if !exists {
		t.Errorf("Should have been considered existing since we're comparing a range array against itself")
	}
	exists = ipPermissionExists(&oldIpPermission, &oldIpPermission, false)
	if !exists {
		t.Errorf("Should have been considered existing since we're comparing a range array against itself")
	}

	exists = ipPermissionExists(&existingIpPermission, &oldIpPermission, false)
	if !exists {
		t.Errorf("Should have been considered existing since 10.* is in oldIpPermission's array of ranges")
	}
	exists = ipPermissionExists(&existingIpPermission2, &oldIpPermission, false)
	if !exists {
		t.Errorf("Should have been considered existing since 192.* is in oldIpPermission2's array of ranges")
	}

	exists = ipPermissionExists(&newIpPermission, &emptyIpPermission, false)
	if exists {
		t.Errorf("Should have not been considered existing since we compared against a missing array of ranges")
	}
	exists = ipPermissionExists(&newIpPermission, &oldIpPermission, false)
	if exists {
		t.Errorf("Should have not been considered existing since 172.* is not in oldIpPermission's array of ranges")
	}
}

func TestIpPermissionExistsHandlesMultipleGroupIdsWithUserIds(t *testing.T) {
	oldIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("firstGroupId"), UserId: aws.String("firstUserId")},
			{GroupId: aws.String("secondGroupId"), UserId: aws.String("secondUserId")},
			{GroupId: aws.String("thirdGroupId"), UserId: aws.String("thirdUserId")},
		},
	}

	existingIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("secondGroupId"), UserId: aws.String("secondUserId")},
		},
	}

	newIpPermission := ec2.IpPermission{
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: aws.String("secondGroupId"), UserId: aws.String("anotherUserId")},
		},
	}

	equals := ipPermissionExists(&existingIpPermission, &oldIpPermission, true)
	if !equals {
		t.Errorf("Should have been considered equal since first is in the second array of groups")
	}

	equals = ipPermissionExists(&newIpPermission, &oldIpPermission, true)
	if equals {
		t.Errorf("Should have not been considered equal since first is not in the second array of groups")
	}
}

func TestFindInstancesByNodeNameCached(t *testing.T) {
	awsServices := NewFakeAWSServices()

	nodeNameOne := "my-dns.internal"
	nodeNameTwo := "my-dns-two.internal"

	var tag ec2.Tag
	tag.Key = aws.String(TagNameKubernetesCluster)
	tag.Value = aws.String(TestClusterId)
	tags := []*ec2.Tag{&tag}

	var runningInstance ec2.Instance
	runningInstance.InstanceId = aws.String("i-running")
	runningInstance.PrivateDnsName = aws.String(nodeNameOne)
	runningInstance.State = &ec2.InstanceState{Code: aws.Int64(16), Name: aws.String("running")}
	runningInstance.Tags = tags

	var secondInstance ec2.Instance

	secondInstance.InstanceId = aws.String("i-running")
	secondInstance.PrivateDnsName = aws.String(nodeNameTwo)
	secondInstance.State = &ec2.InstanceState{Code: aws.Int64(48), Name: aws.String("running")}
	secondInstance.Tags = tags

	var terminatedInstance ec2.Instance
	terminatedInstance.InstanceId = aws.String("i-terminated")
	terminatedInstance.PrivateDnsName = aws.String(nodeNameOne)
	terminatedInstance.State = &ec2.InstanceState{Code: aws.Int64(48), Name: aws.String("terminated")}
	terminatedInstance.Tags = tags

	instances := []*ec2.Instance{&secondInstance, &runningInstance, &terminatedInstance}
	awsServices.instances = append(awsServices.instances, instances...)

	c, err := newAWSCloud(strings.NewReader("[global]"), awsServices)
	if err != nil {
		t.Errorf("Error building aws cloud: %v", err)
		return
	}

	nodeNames := sets.NewString(nodeNameOne)
	returnedInstances, errr := c.getInstancesByNodeNamesCached(nodeNames)

	if errr != nil {
		t.Errorf("Failed to find instance: %v", err)
		return
	}

	if len(returnedInstances) != 1 {
		t.Errorf("Expected a single isntance but found: %v", returnedInstances)
	}

	if *returnedInstances[0].PrivateDnsName != nodeNameOne {
		t.Errorf("Expected node name %v but got %v", nodeNameOne, returnedInstances[0].PrivateDnsName)
	}
}

func (self *FakeELB) expectDescribeLoadBalancers(loadBalancerName string) {
	self.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{LoadBalancerNames: []*string{aws.String(loadBalancerName)}}).Return(&elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{{}},
	})
}
