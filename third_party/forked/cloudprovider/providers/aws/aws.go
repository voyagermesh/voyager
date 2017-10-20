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
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/golang/glog"
	"gopkg.in/gcfg.v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ProviderName is the name of this cloud provider.
const ProviderName = "aws"

// TagNameKubernetesCluster is the tag name we use to differentiate multiple
// logically independent clusters running in the same AZ
const TagNameKubernetesCluster = "KubernetesCluster"

// TagNameVoyagerCluster is Voyager's version of `TagNameKubernetesCluster`. This is kept separate to avoid
// errors like: https://github.com/appscode/voyager/pull/397
const TagNameVoyagerCluster = "VoyagerCluster"

// MaxReadThenCreateRetries sets the maximum number of attempts we will make when
// we read to see if something exists and then try to create it if we didn't find it.
// This can fail once in a consistent system if done in parallel
// In an eventually consistent system, it could fail unboundedly
const MaxReadThenCreateRetries = 30

// Used to call aws_credentials.Init() just once
var once sync.Once

// Services is an abstraction over AWS, to allow mocking/other implementations
type Services interface {
	Compute(region string) (EC2, error)
	LoadBalancing(region string) (ELB, error)
	Metadata() (EC2Metadata, error)
}

// EC2 is an abstraction over AWS', to allow mocking/other implementations
// Note that the DescribeX functions return a list, so callers don't need to deal with paging
// TODO: Should we rename this to AWS (EBS & ELB are not technically part of EC2)
type EC2 interface {
	// Query EC2 for instances matching the filter
	DescribeInstances(request *ec2.DescribeInstancesInput) ([]*ec2.Instance, error)

	DescribeSecurityGroups(request *ec2.DescribeSecurityGroupsInput) ([]*ec2.SecurityGroup, error)

	CreateSecurityGroup(*ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error)
	DeleteSecurityGroup(request *ec2.DeleteSecurityGroupInput) (*ec2.DeleteSecurityGroupOutput, error)

	AuthorizeSecurityGroupIngress(*ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	RevokeSecurityGroupIngress(*ec2.RevokeSecurityGroupIngressInput) (*ec2.RevokeSecurityGroupIngressOutput, error)

	DescribeSubnets(*ec2.DescribeSubnetsInput) ([]*ec2.Subnet, error)

	CreateTags(*ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)

	DescribeRouteTables(request *ec2.DescribeRouteTablesInput) ([]*ec2.RouteTable, error)
	CreateRoute(request *ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error)
	DeleteRoute(request *ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error)

	ModifyInstanceAttribute(request *ec2.ModifyInstanceAttributeInput) (*ec2.ModifyInstanceAttributeOutput, error)
}

// ELB is a simple pass-through of AWS' ELB client interface, which allows for testing
type ELB interface {
	CreateLoadBalancer(*elb.CreateLoadBalancerInput) (*elb.CreateLoadBalancerOutput, error)
	DeleteLoadBalancer(*elb.DeleteLoadBalancerInput) (*elb.DeleteLoadBalancerOutput, error)
	DescribeLoadBalancers(*elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error)
	RegisterInstancesWithLoadBalancer(*elb.RegisterInstancesWithLoadBalancerInput) (*elb.RegisterInstancesWithLoadBalancerOutput, error)
	DeregisterInstancesFromLoadBalancer(*elb.DeregisterInstancesFromLoadBalancerInput) (*elb.DeregisterInstancesFromLoadBalancerOutput, error)
	CreateLoadBalancerPolicy(*elb.CreateLoadBalancerPolicyInput) (*elb.CreateLoadBalancerPolicyOutput, error)
	SetLoadBalancerPoliciesForBackendServer(*elb.SetLoadBalancerPoliciesForBackendServerInput) (*elb.SetLoadBalancerPoliciesForBackendServerOutput, error)

	DetachLoadBalancerFromSubnets(*elb.DetachLoadBalancerFromSubnetsInput) (*elb.DetachLoadBalancerFromSubnetsOutput, error)
	AttachLoadBalancerToSubnets(*elb.AttachLoadBalancerToSubnetsInput) (*elb.AttachLoadBalancerToSubnetsOutput, error)

	CreateLoadBalancerListeners(*elb.CreateLoadBalancerListenersInput) (*elb.CreateLoadBalancerListenersOutput, error)
	DeleteLoadBalancerListeners(*elb.DeleteLoadBalancerListenersInput) (*elb.DeleteLoadBalancerListenersOutput, error)

	ApplySecurityGroupsToLoadBalancer(*elb.ApplySecurityGroupsToLoadBalancerInput) (*elb.ApplySecurityGroupsToLoadBalancerOutput, error)

	ConfigureHealthCheck(*elb.ConfigureHealthCheckInput) (*elb.ConfigureHealthCheckOutput, error)

	DescribeLoadBalancerAttributes(*elb.DescribeLoadBalancerAttributesInput) (*elb.DescribeLoadBalancerAttributesOutput, error)
	ModifyLoadBalancerAttributes(*elb.ModifyLoadBalancerAttributesInput) (*elb.ModifyLoadBalancerAttributesOutput, error)
}

// EC2Metadata is an abstraction over the AWS metadata service.
type EC2Metadata interface {
	// Query the EC2 metadata service (used to discover instance-id etc)
	GetMetadata(path string) (string, error)
}

// InstanceGroupInfo is returned by InstanceGroups.Describe, and exposes information about the group.
type InstanceGroupInfo interface {
	// The number of instances currently running under control of this group
	CurrentSize() (int, error)
}

// Cloud is an implementation of Interface, LoadBalancer and Instances for Amazon Web Services.
type Cloud struct {
	ec2      EC2
	elb      ELB
	metadata EC2Metadata
	cfg      *CloudConfig
	region   string
	vpcID    string

	filterTags map[string]string

	// The AWS instance that we are running on
	// Note that we cache some state in awsInstance (mountpoints), so we must preserve the instance
	selfAWSInstance *awsInstance

	mutex                    sync.Mutex
	lastNodeNames            sets.String
	lastInstancesByNodeNames []*ec2.Instance
}

// CloudConfig wraps the settings for the AWS cloud provider.
type CloudConfig struct {
	Global struct {
		// TODO: Is there any use for this?  We can get it from the instance metadata service
		// Maybe if we're not running on AWS, e.g. bootstrap; for now it is not very useful
		Zone string

		KubernetesClusterTag string

		//The aws provider creates an inbound rule per load balancer on the node security
		//group. However, this can run into the AWS security group rule limit of 50 if
		//many LoadBalancers are created.
		//
		//This flag disables the automatic ingress creation. It requires that the user
		//has setup a rule that allows inbound traffic on kubelet ports from the
		//local VPC subnet (so load balancers can access it). E.g. 10.82.0.0/16 30000-32000.
		DisableSecurityGroupIngress bool

		//During the instantiation of an new AWS cloud provider, the detected region
		//is validated against a known set of regions.
		//
		//In a non-standard, AWS like environment (e.g. Eucalyptus), this check may
		//be undesirable.  Setting this to true will disable the check and provide
		//a warning that the check was skipped.  Please note that this is an
		//experimental feature and work-in-progress for the moment.  If you find
		//yourself in an non-AWS cloud and open an issue, please indicate that in the
		//issue body.
		DisableStrictZoneCheck bool
	}
}

// awsSdkEC2 is an implementation of the EC2 interface, backed by aws-sdk-go
type awsSdkEC2 struct {
	ec2 *ec2.EC2
}

type awsSDKProvider struct {
	creds *credentials.Credentials

	mutex          sync.Mutex
	regionDelayers map[string]*CrossRequestRetryDelay
}

func newAWSSDKProvider(creds *credentials.Credentials) *awsSDKProvider {
	return &awsSDKProvider{
		creds:          creds,
		regionDelayers: make(map[string]*CrossRequestRetryDelay),
	}
}

func (p *awsSDKProvider) addHandlers(regionName string, h *request.Handlers) {
	h.Sign.PushFrontNamed(request.NamedHandler{
		Name: "k8s/logger",
		Fn:   awsHandlerLogger,
	})

	delayer := p.getCrossRequestRetryDelay(regionName)
	if delayer != nil {
		h.Sign.PushFrontNamed(request.NamedHandler{
			Name: "k8s/delay-presign",
			Fn:   delayer.BeforeSign,
		})

		h.AfterRetry.PushFrontNamed(request.NamedHandler{
			Name: "k8s/delay-afterretry",
			Fn:   delayer.AfterRetry,
		})
	}
}

// Get a CrossRequestRetryDelay, scoped to the region, not to the request.
// This means that when we hit a limit on a call, we will delay _all_ calls to the API.
// We do this to protect the AWS account from becoming overloaded and effectively locked.
// We also log when we hit request limits.
// Note that this delays the current goroutine; this is bad behaviour and will
// likely cause k8s to become slow or unresponsive for cloud operations.
// However, this throttle is intended only as a last resort.  When we observe
// this throttling, we need to address the root cause (e.g. add a delay to a
// controller retry loop)
func (p *awsSDKProvider) getCrossRequestRetryDelay(regionName string) *CrossRequestRetryDelay {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delayer, found := p.regionDelayers[regionName]
	if !found {
		delayer = NewCrossRequestRetryDelay()
		p.regionDelayers[regionName] = delayer
	}
	return delayer
}

func (p *awsSDKProvider) Compute(regionName string) (EC2, error) {
	service := ec2.New(session.New(&aws.Config{
		Region:      &regionName,
		Credentials: p.creds,
	}))

	p.addHandlers(regionName, &service.Handlers)

	ec2 := &awsSdkEC2{
		ec2: service,
	}
	return ec2, nil
}

func (p *awsSDKProvider) LoadBalancing(regionName string) (ELB, error) {
	elbClient := elb.New(session.New(&aws.Config{
		Region:      &regionName,
		Credentials: p.creds,
	}))

	p.addHandlers(regionName, &elbClient.Handlers)

	return elbClient, nil
}

func (p *awsSDKProvider) Metadata() (EC2Metadata, error) {
	client := ec2metadata.New(session.New(&aws.Config{}))
	return client, nil
}

// isNilOrEmpty returns true if the value is nil or ""
// Deprecated: prefer aws.StringValue(x) == "" (and elimination of this check altogether whrere possible)
func isNilOrEmpty(s *string) bool {
	return s == nil || *s == ""
}

// orEmpty returns the string value, or "" if the pointer is nil
// Deprecated: prefer aws.StringValue
func orEmpty(s *string) string {
	return aws.StringValue(s)
}

func newEc2Filter(name string, value string) *ec2.Filter {
	filter := &ec2.Filter{
		Name: aws.String(name),
		Values: []*string{
			aws.String(value),
		},
	}
	return filter
}

// AddSSHKeyToAllInstances is currently not implemented.
func (c *Cloud) AddSSHKeyToAllInstances(user string, keyData []byte) error {
	return errors.New("unimplemented")
}

// CurrentNodeName returns the name of the current node
func (c *Cloud) CurrentNodeName(hostname string) (types.NodeName, error) {
	return c.selfAWSInstance.nodeName, nil
}

// Implementation of EC2.Instances
func (s *awsSdkEC2) DescribeInstances(request *ec2.DescribeInstancesInput) ([]*ec2.Instance, error) {
	// Instances are paged
	results := []*ec2.Instance{}
	var nextToken *string

	for {
		response, err := s.ec2.DescribeInstances(request)
		if err != nil {
			return nil, fmt.Errorf("error listing AWS instances: %v", err)
		}

		for _, reservation := range response.Reservations {
			results = append(results, reservation.Instances...)
		}

		nextToken = response.NextToken
		if isNilOrEmpty(nextToken) {
			break
		}
		request.NextToken = nextToken
	}

	return results, nil
}

// Implements EC2.DescribeSecurityGroups
func (s *awsSdkEC2) DescribeSecurityGroups(request *ec2.DescribeSecurityGroupsInput) ([]*ec2.SecurityGroup, error) {
	// Security groups are not paged
	response, err := s.ec2.DescribeSecurityGroups(request)
	if err != nil {
		return nil, fmt.Errorf("error listing AWS security groups: %v", err)
	}
	return response.SecurityGroups, nil
}

func (s *awsSdkEC2) AttachVolume(request *ec2.AttachVolumeInput) (*ec2.VolumeAttachment, error) {
	return s.ec2.AttachVolume(request)
}

func (s *awsSdkEC2) DetachVolume(request *ec2.DetachVolumeInput) (*ec2.VolumeAttachment, error) {
	return s.ec2.DetachVolume(request)
}

func (s *awsSdkEC2) DescribeVolumes(request *ec2.DescribeVolumesInput) ([]*ec2.Volume, error) {
	// Volumes are paged
	results := []*ec2.Volume{}
	var nextToken *string

	for {
		response, err := s.ec2.DescribeVolumes(request)

		if err != nil {
			return nil, fmt.Errorf("error listing AWS volumes: %v", err)
		}

		results = append(results, response.Volumes...)

		nextToken = response.NextToken
		if isNilOrEmpty(nextToken) {
			break
		}
		request.NextToken = nextToken
	}

	return results, nil
}

func (s *awsSdkEC2) CreateVolume(request *ec2.CreateVolumeInput) (resp *ec2.Volume, err error) {
	return s.ec2.CreateVolume(request)
}

func (s *awsSdkEC2) DeleteVolume(request *ec2.DeleteVolumeInput) (*ec2.DeleteVolumeOutput, error) {
	return s.ec2.DeleteVolume(request)
}

func (s *awsSdkEC2) DescribeSubnets(request *ec2.DescribeSubnetsInput) ([]*ec2.Subnet, error) {
	// Subnets are not paged
	response, err := s.ec2.DescribeSubnets(request)
	if err != nil {
		return nil, fmt.Errorf("error listing AWS subnets: %v", err)
	}
	return response.Subnets, nil
}

func (s *awsSdkEC2) CreateSecurityGroup(request *ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error) {
	return s.ec2.CreateSecurityGroup(request)
}

func (s *awsSdkEC2) DeleteSecurityGroup(request *ec2.DeleteSecurityGroupInput) (*ec2.DeleteSecurityGroupOutput, error) {
	return s.ec2.DeleteSecurityGroup(request)
}

func (s *awsSdkEC2) AuthorizeSecurityGroupIngress(request *ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	return s.ec2.AuthorizeSecurityGroupIngress(request)
}

func (s *awsSdkEC2) RevokeSecurityGroupIngress(request *ec2.RevokeSecurityGroupIngressInput) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	return s.ec2.RevokeSecurityGroupIngress(request)
}

func (s *awsSdkEC2) CreateTags(request *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return s.ec2.CreateTags(request)
}

func (s *awsSdkEC2) DescribeRouteTables(request *ec2.DescribeRouteTablesInput) ([]*ec2.RouteTable, error) {
	// Not paged
	response, err := s.ec2.DescribeRouteTables(request)
	if err != nil {
		return nil, fmt.Errorf("error listing AWS route tables: %v", err)
	}
	return response.RouteTables, nil
}

func (s *awsSdkEC2) CreateRoute(request *ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error) {
	return s.ec2.CreateRoute(request)
}

func (s *awsSdkEC2) DeleteRoute(request *ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error) {
	return s.ec2.DeleteRoute(request)
}

func (s *awsSdkEC2) ModifyInstanceAttribute(request *ec2.ModifyInstanceAttributeInput) (*ec2.ModifyInstanceAttributeOutput, error) {
	return s.ec2.ModifyInstanceAttribute(request)
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		creds := credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&ec2rolecreds.EC2RoleProvider{
					Client: ec2metadata.New(session.New(&aws.Config{})),
				},
				&credentials.SharedCredentialsProvider{},
			})
		aws := newAWSSDKProvider(creds)
		return newAWSCloud(config, aws)
	})
}

// readAWSCloudConfig reads an instance of AWSCloudConfig from config reader.
func readAWSCloudConfig(config io.Reader, metadata EC2Metadata) (*CloudConfig, error) {
	var cfg CloudConfig
	var err error

	if config != nil {
		err = gcfg.ReadInto(&cfg, config)
		if err != nil {
			return nil, err
		}
	}

	if cfg.Global.Zone == "" {
		if metadata != nil {
			glog.Info("Zone not specified in configuration file; querying AWS metadata service")
			cfg.Global.Zone, err = getAvailabilityZone(metadata)
			if err != nil {
				return nil, err
			}
		}
		if cfg.Global.Zone == "" {
			return nil, fmt.Errorf("no zone specified in configuration file")
		}
	}

	return &cfg, nil
}

func getAvailabilityZone(metadata EC2Metadata) (string, error) {
	return metadata.GetMetadata("placement/availability-zone")
}

// Derives the region from a valid az name.
// Returns an error if the az is known invalid (empty)
func azToRegion(az string) (string, error) {
	if len(az) < 1 {
		return "", fmt.Errorf("invalid (empty) AZ")
	}
	region := az[:len(az)-1]
	return region, nil
}

// newAWSCloud creates a new instance of AWSCloud.
// AWSProvider and instanceId are primarily for tests
func newAWSCloud(config io.Reader, awsServices Services) (*Cloud, error) {
	// We have some state in the Cloud object - in particular the attaching map
	// Log so that if we are building multiple Cloud objects, it is obvious!
	glog.Infof("Building AWS cloudprovider")

	// Register handler for ECR credentials
	// Register regions, in particular for ECR credentials
	once.Do(func() {
		RecognizeWellKnownRegions()
	})

	metadata, err := awsServices.Metadata()
	if err != nil {
		return nil, fmt.Errorf("error creating AWS metadata client: %v", err)
	}

	cfg, err := readAWSCloudConfig(config, metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to read AWS cloud provider config file: %v", err)
	}

	zone := cfg.Global.Zone
	if len(zone) <= 1 {
		return nil, fmt.Errorf("invalid AWS zone in config file: %s", zone)
	}
	regionName, err := azToRegion(zone)
	if err != nil {
		return nil, err
	}

	if !cfg.Global.DisableStrictZoneCheck {
		valid := isRegionValid(regionName)
		if !valid {
			return nil, fmt.Errorf("not a valid AWS zone (unknown region): %s", zone)
		}
	} else {
		glog.Warningf("Strict AWS zone checking is disabled.  Proceeding with zone: %s", zone)
	}

	ec2, err := awsServices.Compute(regionName)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS EC2 client: %v", err)
	}

	elb, err := awsServices.LoadBalancing(regionName)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS ELB client: %v", err)
	}

	awsCloud := &Cloud{
		ec2:      ec2,
		elb:      elb,
		metadata: metadata,
		cfg:      cfg,
		region:   regionName,
	}

	selfAWSInstance, err := awsCloud.buildSelfAWSInstance()
	if err != nil {
		return nil, err
	}

	awsCloud.selfAWSInstance = selfAWSInstance
	awsCloud.vpcID = selfAWSInstance.vpcID

	filterTags := map[string]string{}
	if cfg.Global.KubernetesClusterTag != "" {
		filterTags[TagNameKubernetesCluster] = cfg.Global.KubernetesClusterTag
	} else {
		// TODO: Clean up double-API query
		info, err := selfAWSInstance.describeInstance()
		if err != nil {
			return nil, err
		}
		for _, tag := range info.Tags {
			if orEmpty(tag.Key) == TagNameKubernetesCluster {
				filterTags[TagNameKubernetesCluster] = orEmpty(tag.Value)
			}
		}
	}

	if filterTags[TagNameKubernetesCluster] == "" {
		glog.Errorf("Tag %q not found; Kubernetes may behave unexpectedly.", TagNameKubernetesCluster)
	}

	awsCloud.filterTags = filterTags
	if len(filterTags) > 0 {
		glog.Infof("AWS cloud filtering on tags: %v", filterTags)
	} else {
		glog.Infof("AWS cloud - no tag filtering")
	}

	return awsCloud, nil
}

// ProviderName returns the cloud provider ID.
func (c *Cloud) ProviderName() string {
	return ProviderName
}

// Firewall returns an implementation of Firewall for Amazon Web Services.
func (c *Cloud) Firewall() (cloudprovider.Firewall, bool) {
	return c, true
}

// Abstraction around AWS Instance Types
// There isn't an API to get information for a particular instance type (that I know of)
type awsInstanceType struct {
}

// Used to represent a mount device for attaching an EBS volume
// This should be stored as a single letter (i.e. c, not sdc or /dev/sdc)
type mountDevice string

type awsInstance struct {
	ec2 EC2

	// id in AWS
	awsID string

	// node name in k8s
	nodeName types.NodeName

	// availability zone the instance resides in
	availabilityZone string

	// ID of VPC the instance resides in
	vpcID string

	// ID of subnet the instance resides in
	subnetID string

	// instance type
	instanceType string
}

// newAWSInstance creates a new awsInstance object
func newAWSInstance(ec2Service EC2, instance *ec2.Instance) *awsInstance {
	az := ""
	if instance.Placement != nil {
		az = aws.StringValue(instance.Placement.AvailabilityZone)
	}
	self := &awsInstance{
		ec2:              ec2Service,
		awsID:            aws.StringValue(instance.InstanceId),
		nodeName:         mapInstanceToNodeName(instance),
		availabilityZone: az,
		instanceType:     aws.StringValue(instance.InstanceType),
		vpcID:            aws.StringValue(instance.VpcId),
		subnetID:         aws.StringValue(instance.SubnetId),
	}

	return self
}

// Gets the awsInstanceType that models the instance type of this instance
func (i *awsInstance) getInstanceType() *awsInstanceType {
	// TODO: Make this real
	awsInstanceType := &awsInstanceType{}
	return awsInstanceType
}

// Gets the full information about this instance from the EC2 API
func (i *awsInstance) describeInstance() (*ec2.Instance, error) {
	instanceID := i.awsID
	request := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{&instanceID},
	}

	instances, err := i.ec2.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for instance: %s", i.awsID)
	}
	if len(instances) > 1 {
		return nil, fmt.Errorf("multiple instances found for instance: %s", i.awsID)
	}
	return instances[0], nil
}

// Builds the awsInstance for the EC2 instance on which we are running.
// This is called when the AWSCloud is initialized, and should not be called otherwise (because the awsInstance for the local instance is a singleton with drive mapping state)
func (c *Cloud) buildSelfAWSInstance() (*awsInstance, error) {
	if c.selfAWSInstance != nil {
		panic("do not call buildSelfAWSInstance directly")
	}
	instanceID, err := c.metadata.GetMetadata("instance-id")
	if err != nil {
		return nil, fmt.Errorf("error fetching instance-id from ec2 metadata service: %v", err)
	}

	// We want to fetch the hostname via the EC2 metadata service
	// (`GetMetadata("local-hostname")`): But see #11543 - we need to use
	// the EC2 API to get the privateDnsName in case of a private DNS zone
	// e.g. mydomain.io, because the metadata service returns the wrong
	// hostname.  Once we're doing that, we might as well get all our
	// information from the instance returned by the EC2 API - it is a
	// single API call to get all the information, and it means we don't
	// have two code paths.
	instance, err := c.getInstanceByID(instanceID)
	if err != nil {
		return nil, fmt.Errorf("error finding instance %s: %v", instanceID, err)
	}
	return newAWSInstance(c.ec2, instance), nil
}

// Retrieves the specified security group from the AWS API, or returns nil if not found
func (c *Cloud) findSecurityGroup(securityGroupID string) (*ec2.SecurityGroup, error) {
	describeSecurityGroupsRequest := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{&securityGroupID},
	}
	// We don't apply our tag filters because we are retrieving by ID

	groups, err := c.ec2.DescribeSecurityGroups(describeSecurityGroupsRequest)
	if err != nil {
		glog.Warningf("Error retrieving security group: %q", err)
		return nil, err
	}

	if len(groups) == 0 {
		return nil, nil
	}
	if len(groups) != 1 {
		// This should not be possible - ids should be unique
		return nil, fmt.Errorf("multiple security groups found with same id %q", securityGroupID)
	}
	group := groups[0]
	return group, nil
}

func isEqualIntPointer(l, r *int64) bool {
	if l == nil {
		return r == nil
	}
	if r == nil {
		return l == nil
	}
	return *l == *r
}

func isEqualStringPointer(l, r *string) bool {
	if l == nil {
		return r == nil
	}
	if r == nil {
		return l == nil
	}
	return *l == *r
}

func ipPermissionExists(newPermission, existing *ec2.IpPermission, compareGroupUserIDs bool) bool {
	if !isEqualIntPointer(newPermission.FromPort, existing.FromPort) {
		return false
	}
	if !isEqualIntPointer(newPermission.ToPort, existing.ToPort) {
		return false
	}
	if !isEqualStringPointer(newPermission.IpProtocol, existing.IpProtocol) {
		return false
	}
	// Check only if newPermission is a subset of existing. Usually it has zero or one elements.
	// Not doing actual CIDR math yet; not clear it's needed, either.
	glog.V(4).Infof("Comparing %v to %v", newPermission, existing)
	if len(newPermission.IpRanges) > len(existing.IpRanges) {
		return false
	}

	for j := range newPermission.IpRanges {
		found := false
		for k := range existing.IpRanges {
			if isEqualStringPointer(newPermission.IpRanges[j].CidrIp, existing.IpRanges[k].CidrIp) {
				found = true
				break
			}
		}
		if found == false {
			return false
		}
	}
	for _, leftPair := range newPermission.UserIdGroupPairs {
		for _, rightPair := range existing.UserIdGroupPairs {
			if isEqualUserGroupPair(leftPair, rightPair, compareGroupUserIDs) {
				return true
			}
		}
		return false
	}

	return true
}

func isEqualUserGroupPair(l, r *ec2.UserIdGroupPair, compareGroupUserIDs bool) bool {
	glog.V(2).Infof("Comparing %v to %v", *l.GroupId, *r.GroupId)
	if isEqualStringPointer(l.GroupId, r.GroupId) {
		if compareGroupUserIDs {
			if isEqualStringPointer(l.UserId, r.UserId) {
				return true
			}
		} else {
			return true
		}
	}

	return false
}

// Makes sure the security group ingress is exactly the specified permissions
// Returns true if and only if changes were made
// The security group must already exist
func (c *Cloud) setSecurityGroupIngress(securityGroupID string, permissions IPPermissionSet) (bool, error) {
	group, err := c.findSecurityGroup(securityGroupID)
	if err != nil {
		glog.Warning("Error retrieving security group", err)
		return false, err
	}

	if group == nil {
		return false, fmt.Errorf("security group not found: %s", securityGroupID)
	}

	glog.V(2).Infof("Existing security group ingress: %s %v", securityGroupID, group.IpPermissions)

	actual := NewIPPermissionSet(group.IpPermissions...)

	// EC2 groups rules together, for example combining:
	//
	// { Port=80, Range=[A] } and { Port=80, Range=[B] }
	//
	// into { Port=80, Range=[A,B] }
	//
	// We have to ungroup them, because otherwise the logic becomes really
	// complicated, and also because if we have Range=[A,B] and we try to
	// add Range=[A] then EC2 complains about a duplicate rule.
	permissions = permissions.Ungroup()
	actual = actual.Ungroup()

	remove := actual.Difference(permissions)
	add := permissions.Difference(actual)

	if add.Len() == 0 && remove.Len() == 0 {
		return false, nil
	}

	// TODO: There is a limit in VPC of 100 rules per security group, so we
	// probably should try grouping or combining to fit under this limit.
	// But this is only used on the ELB security group currently, so it
	// would require (ports * CIDRS) > 100.  Also, it isn't obvious exactly
	// how removing single permissions from compound rules works, and we
	// don't want to accidentally open more than intended while we're
	// applying changes.
	if add.Len() != 0 {
		glog.V(2).Infof("Adding security group ingress: %s %v", securityGroupID, add.List())

		request := &ec2.AuthorizeSecurityGroupIngressInput{}
		request.GroupId = &securityGroupID
		request.IpPermissions = add.List()
		_, err = c.ec2.AuthorizeSecurityGroupIngress(request)
		if err != nil {
			return false, fmt.Errorf("error authorizing security group ingress: %v", err)
		}
	}
	if remove.Len() != 0 {
		glog.V(2).Infof("Remove security group ingress: %s %v", securityGroupID, remove.List())

		request := &ec2.RevokeSecurityGroupIngressInput{}
		request.GroupId = &securityGroupID
		request.IpPermissions = remove.List()
		_, err = c.ec2.RevokeSecurityGroupIngress(request)
		if err != nil {
			return false, fmt.Errorf("error revoking security group ingress: %v", err)
		}
	}

	return true, nil
}

// Makes sure the security group includes the specified permissions
// Returns true if and only if changes were made
// The security group must already exist
func (c *Cloud) addSecurityGroupIngress(securityGroupID string, addPermissions []*ec2.IpPermission) (bool, error) {
	group, err := c.findSecurityGroup(securityGroupID)
	if err != nil {
		glog.Warningf("Error retrieving security group: %v", err)
		return false, err
	}

	if group == nil {
		return false, fmt.Errorf("security group not found: %s", securityGroupID)
	}

	glog.V(2).Infof("Existing security group ingress: %s %v", securityGroupID, group.IpPermissions)

	changes := []*ec2.IpPermission{}
	for _, addPermission := range addPermissions {
		hasUserID := false
		for i := range addPermission.UserIdGroupPairs {
			if addPermission.UserIdGroupPairs[i].UserId != nil {
				hasUserID = true
			}
		}

		found := false
		for _, groupPermission := range group.IpPermissions {
			if ipPermissionExists(addPermission, groupPermission, hasUserID) {
				found = true
				break
			}
		}

		if !found {
			changes = append(changes, addPermission)
		}
	}

	if len(changes) == 0 {
		return false, nil
	}

	glog.V(2).Infof("Adding security group ingress: %s %v", securityGroupID, changes)

	request := &ec2.AuthorizeSecurityGroupIngressInput{}
	request.GroupId = &securityGroupID
	request.IpPermissions = changes
	_, err = c.ec2.AuthorizeSecurityGroupIngress(request)
	if err != nil {
		glog.Warning("Error authorizing security group ingress", err)
		return false, fmt.Errorf("error authorizing security group ingress: %v", err)
	}

	return true, nil
}

// Makes sure the security group no longer includes the specified permissions
// Returns true if and only if changes were made
// If the security group no longer exists, will return (false, nil)
func (c *Cloud) removeSecurityGroupIngress(securityGroupID string, removePermissions []*ec2.IpPermission) (bool, error) {
	group, err := c.findSecurityGroup(securityGroupID)
	if err != nil {
		glog.Warningf("Error retrieving security group: %v", err)
		return false, err
	}

	if group == nil {
		glog.Warning("Security group not found: ", securityGroupID)
		return false, nil
	}

	changes := []*ec2.IpPermission{}
	for _, removePermission := range removePermissions {
		hasUserID := false
		for i := range removePermission.UserIdGroupPairs {
			if removePermission.UserIdGroupPairs[i].UserId != nil {
				hasUserID = true
			}
		}

		var found *ec2.IpPermission
		for _, groupPermission := range group.IpPermissions {
			if ipPermissionExists(removePermission, groupPermission, hasUserID) {
				found = removePermission
				break
			}
		}

		if found != nil {
			changes = append(changes, found)
		}
	}

	if len(changes) == 0 {
		return false, nil
	}

	glog.V(2).Infof("Removing security group ingress: %s %v", securityGroupID, changes)

	request := &ec2.RevokeSecurityGroupIngressInput{}
	request.GroupId = &securityGroupID
	request.IpPermissions = changes
	_, err = c.ec2.RevokeSecurityGroupIngress(request)
	if err != nil {
		glog.Warningf("Error revoking security group ingress: %v", err)
		return false, err
	}

	return true, nil
}

// Ensure that a resource has the correct tags
// If it has no tags, we assume that this was a problem caused by an error in between creation and tagging,
// and we add the tags.  If it has a different cluster's tags, that is an error.
func (c *Cloud) ensureVoyagerTags(resourceID string, tags []*ec2.Tag) error {
	actualTags := make(map[string]string)
	for _, tag := range tags {
		actualTags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}

	clusterTags := map[string]string{
		TagNameVoyagerCluster: c.getClusterName(),
	}
	addTags := make(map[string]string)
	for k, expected := range clusterTags {
		actual := actualTags[k]
		if actual == expected {
			continue
		}
		if actual == "" {
			glog.Warningf("Resource %q was missing expected cluster tag %q.  Will add (with value %q)", resourceID, k, expected)
			addTags[k] = expected
		} else {
			return fmt.Errorf("resource %q has tag belonging to another cluster: %q=%q (expected %q)", resourceID, k, actual, expected)
		}
	}

	if err := c.createTags(resourceID, addTags); err != nil {
		return fmt.Errorf("error adding missing tags to resource %q: %v", resourceID, err)
	}

	return nil
}

// Makes sure the security group exists.
// For multi-cluster isolation, name must be globally unique, for example derived from the service UUID.
// Returns the security group id or error
// Makes sure the security group exists.
// For multi-cluster isolation, name must be globally unique, for example derived from the service UUID.
// Returns the security group id or error
func (c *Cloud) ensureSecurityGroup(name string, description string) (string, error) {
	groupID := ""
	attempt := 0
	for {
		attempt++

		request := &ec2.DescribeSecurityGroupsInput{}
		filters := []*ec2.Filter{
			newEc2Filter("group-name", name),
			newEc2Filter("vpc-id", c.vpcID),
		}
		// Note that we do _not_ add our tag filters; group-name + vpc-id is the EC2 primary key.
		// However, we do check that it matches our tags.
		// If it doesn't have any tags, we tag it; this is how we recover if we failed to tag before.
		// If it has a different cluster's tags, that is an error.
		// This shouldn't happen because name is expected to be globally unique (UUID derived)
		request.Filters = filters

		securityGroups, err := c.ec2.DescribeSecurityGroups(request)
		if err != nil {
			return "", err
		}

		if len(securityGroups) >= 1 {
			if len(securityGroups) > 1 {
				glog.Warningf("Found multiple security groups with name: %q", name)
			}
			err := c.ensureVoyagerTags(aws.StringValue(securityGroups[0].GroupId), securityGroups[0].Tags)
			if err != nil {
				return "", err
			}

			return aws.StringValue(securityGroups[0].GroupId), nil
		}

		createRequest := &ec2.CreateSecurityGroupInput{}
		createRequest.VpcId = &c.vpcID
		createRequest.GroupName = &name
		createRequest.Description = &description

		createResponse, err := c.ec2.CreateSecurityGroup(createRequest)
		if err != nil {
			ignore := false
			switch err := err.(type) {
			case awserr.Error:
				if err.Code() == "InvalidGroup.Duplicate" && attempt < MaxReadThenCreateRetries {
					glog.V(2).Infof("Got InvalidGroup.Duplicate while creating security group (race?); will retry")
					ignore = true
				}
			}
			if !ignore {
				glog.Error("Error creating security group: ", err)
				return "", err
			}
			time.Sleep(1 * time.Second)
		} else {
			groupID = orEmpty(createResponse.GroupId)
			break
		}
	}
	if groupID == "" {
		return "", fmt.Errorf("created security group, but id was not returned: %s", name)
	}

	err := c.ensureVoyagerTags(groupID, nil)
	if err != nil {
		// If we retry, ensureClusterTags will recover from this - it
		// will add the missing tags.  We could delete the security
		// group here, but that doesn't feel like the right thing, as
		// the caller is likely to retry the create
		return "", fmt.Errorf("error tagging security group: %v", err)
	}
	return groupID, nil
}

// createTags calls EC2 CreateTags, but adds retry-on-failure logic
// We retry mainly because if we create an object, we cannot tag it until it is "fully created" (eventual consistency)
// The error code varies though (depending on what we are tagging), so we simply retry on all errors
func (c *Cloud) createTags(resourceID string, tags map[string]string) error {
	if tags == nil || len(tags) == 0 {
		return nil
	}

	var awsTags []*ec2.Tag
	for k, v := range tags {
		tag := &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		awsTags = append(awsTags, tag)
	}

	request := &ec2.CreateTagsInput{}
	request.Resources = []*string{&resourceID}
	request.Tags = awsTags

	// TODO: We really should do exponential backoff here
	attempt := 0
	maxAttempts := 60

	for {
		_, err := c.ec2.CreateTags(request)
		if err == nil {
			return nil
		}

		// We could check that the error is retryable, but the error code changes based on what we are tagging
		// SecurityGroup: InvalidGroup.NotFound
		attempt++
		if attempt > maxAttempts {
			glog.Warningf("Failed to create tags (too many attempts): %v", err)
			return err
		}
		glog.V(2).Infof("Failed to create tags; will retry.  Error was %v", err)
		time.Sleep(1 * time.Second)
	}
}

type portSets struct {
	names   sets.String
	numbers sets.Int64
}

func (c *Cloud) GetSecurityGroupName(service *apiv1.Service) string {
	ret := service.Name + "@" + service.Namespace + "@" + c.getClusterName()
	//AWS requires that the name of a load balancer is shorter than 32 bytes.
	if len(ret) > 32 {
		ret = ret[:32]
	}
	return ret
}

// EnsureFirewall implements Firewall.EnsureFirewall
func (c *Cloud) EnsureFirewall(apiService *apiv1.Service, hostnames []string) error {
	glog.V(2).Infof("EnsureFirewall(%v, %v, %v, %v, %v)",
		apiService.Namespace, apiService.Name, c.region, apiService.Spec.Ports, hostnames)

	if apiService.Spec.SessionAffinity != apiv1.ServiceAffinityNone {
		// ELB supports sticky sessions, but only when configured for HTTP/HTTPS
		return fmt.Errorf("unsupported service affinity: %v", apiService.Spec.SessionAffinity)
	}

	if len(apiService.Spec.Ports) == 0 {
		return errors.New("requested security group with no ports")
	}

	hostSet := sets.NewString(hostnames...)
	instances, err := c.getInstancesByNodeNamesCached(hostSet)
	if err != nil {
		return err
	}

	sourceRanges, err := cloudprovider.GetLoadBalancerSourceRanges(apiService)
	if err != nil {
		return err
	}

	//loadBalancerName := cloudprovider.GetLoadBalancerName(apiService)
	serviceName := types.NamespacedName{Namespace: apiService.Namespace, Name: apiService.Name}

	// Create a security group for the load balancer
	var securityGroupID string
	{
		sgName := c.GetSecurityGroupName(apiService)
		sgDescription := fmt.Sprintf("Security group for Voyager HostPort Ingress %v", serviceName)
		securityGroupID, err = c.ensureSecurityGroup(sgName, sgDescription)
		if err != nil {
			glog.Error("Error creating Voyager security group: ", err)
			return err
		}
		ec2SourceRanges := []*ec2.IpRange{}
		for _, sourceRange := range sourceRanges.StringSlice() {
			ec2SourceRanges = append(ec2SourceRanges, &ec2.IpRange{CidrIp: aws.String(sourceRange)})
		}

		permissions := NewIPPermissionSet()
		for _, port := range apiService.Spec.Ports {
			var portInt64 int64
			if apiService.Spec.Type == apiv1.ServiceTypeNodePort {
				if port.NodePort == 0 {
					glog.Errorf("Ignoring port without NodePort defined: %v", port)
					continue
				}
				portInt64 = int64(port.NodePort)
			} else {
				portInt64 = int64(port.Port)
			}
			protocol := strings.ToLower(string(port.Protocol))

			permission := &ec2.IpPermission{}
			permission.FromPort = &portInt64
			permission.ToPort = &portInt64
			permission.IpRanges = ec2SourceRanges
			permission.IpProtocol = &protocol

			permissions.Insert(permission)
		}
		_, err = c.setSecurityGroupIngress(securityGroupID, permissions)
		if err != nil {
			return err
		}
	}

	err = c.updateInstanceSecurityGroups(securityGroupID, instances)
	if err != nil {
		glog.Warning("Error opening firewall of the instances: ", err)
		return err
	}

	// TODO: Wait for creation?
	return nil
}

// ingressSecurityGroupId
func (c *Cloud) updateInstanceSecurityGroups(ingressSecurityGroupId string, instances []*ec2.Instance) error {
	hostSet := sets.NewString()
	for _, instance := range instances {
		hostSet.Insert(*instance.PrivateDnsName)
	}

	{
		filters := []*ec2.Filter{
			newEc2Filter("instance.group-id", ingressSecurityGroupId),
			newEc2Filter("tag:"+TagNameKubernetesCluster, c.getClusterName()),
		}
		request := &ec2.DescribeInstancesInput{
			Filters: filters,
		}

		exposedInstances, err := c.ec2.DescribeInstances(request)
		if err != nil {
			glog.Warningf("error querying instances with security group %v: %v", ingressSecurityGroupId, err)
			return err
		}
		for _, instance := range exposedInstances {
			if instance.PrivateDnsName == nil || !hostSet.Has(*instance.PrivateDnsName) {
				glog.Infof("Removing voyager security group %s from instance %s", ingressSecurityGroupId, *instance.PrivateDnsName)
				// Remove Ingress SG from remaining instances
				attrRequest := &ec2.ModifyInstanceAttributeInput{}
				attrRequest.InstanceId = instance.InstanceId
				for _, sg := range instance.SecurityGroups {
					if sg.GroupId != nil && *sg.GroupId == ingressSecurityGroupId {
						continue
					}
					attrRequest.Groups = append(attrRequest.Groups, sg.GroupId)
				}
				_, err := c.ec2.ModifyInstanceAttribute(attrRequest)
				if err != nil {
					glog.Warning("Error adding security group to the instance: ", instance.InstanceId, err)
					return err
				}

			}
		}
	}

	{
		// Add to network interface
		for _, instance := range instances {
			glog.Infof("Adding voyager security group %s to instance %s", ingressSecurityGroupId, *instance.PrivateDnsName)
			// Get the actual list of groups that allow ingress from the load-balancer
			attrRequest := &ec2.ModifyInstanceAttributeInput{}
			attrRequest.InstanceId = instance.InstanceId
			attrRequest.Groups = []*string{aws.String(ingressSecurityGroupId)}
			for _, sg := range instance.SecurityGroups {
				attrRequest.Groups = append(attrRequest.Groups, sg.GroupId)
			}
			_, err := c.ec2.ModifyInstanceAttribute(attrRequest)
			if err != nil {
				glog.Warning("Error adding security group to the instance: ", instance.InstanceId, err)
				return err
			}
		}
	}

	return nil
}

// EnsureFirewallDeleted implements Firewall.EnsureFirewallDeleted.
func (c *Cloud) EnsureFirewallDeleted(service *apiv1.Service) error {
	//loadBalancerName := cloudprovider.GetLoadBalancerName(service)
	// Collect the security groups to delete
	var securityGroupID string

	{
		// Delete the security group(s) for the load balancer
		// Note that this is annoying: the load balancer disappears from the API immediately, but it is still
		// deleting in the background.  We get a DependencyViolation until the load balancer has deleted itself

		sgName := c.GetSecurityGroupName(service)

		filters := []*ec2.Filter{
			newEc2Filter("vpc-id", c.vpcID),
			newEc2Filter("group-name", sgName),
			newEc2Filter("tag:"+TagNameVoyagerCluster, c.getClusterName()),
		}
		request := &ec2.DescribeSecurityGroupsInput{
			Filters: filters,
		}
		glog.V(3).Infof("[%s/%s]: Looking up security group %v to delete", service.Namespace, service.Name, request)
		securityGroups, err := c.ec2.DescribeSecurityGroups(request)
		glog.V(3).Infof("[%s/%s]: Found security group: %v", service.Namespace, service.Name, securityGroups)
		if err != nil {
			ignore := false
			if awsError, ok := err.(awserr.Error); ok {
				if awsError.Code() == "DependencyViolation" {
					glog.V(2).Infof("Ignoring DependencyViolation while describing ingress security group (%s), assuming because Voyager security group is in process of deleting", securityGroupID)
					ignore = true
				}
			}
			if !ignore {
				return fmt.Errorf("error while describing load balancer security group (%s): %v", sgName, err)
			}
		}
		if len(securityGroups) > 1 {
			return fmt.Errorf("Multiple securitygroup found when EnsureFirewallDeleted for service %v", sgName)
		}
		for _, securityGroup := range securityGroups {
			securityGroupID = *securityGroup.GroupId
			break
		}
	}

	{
		// De-authorize the ingress security group from the instances security group
		err := c.updateInstanceSecurityGroups(securityGroupID, nil)
		if err != nil {
			glog.Error("Error deregistering voyager security group from instance security groups: ", err)
			return err
		}
	}

	attempt := 0
	for {
		request := &ec2.DeleteSecurityGroupInput{}
		request.GroupId = &securityGroupID
		_, err := c.ec2.DeleteSecurityGroup(request)
		if err != nil {
			ignore := false
			if awsError, ok := err.(awserr.Error); ok {
				if awsError.Code() == "DependencyViolation" {
					glog.V(2).Infof("[%s/%s] Attempt %d: Ignoring DependencyViolation while deleting ingress security group (%s), assuming because SG is in process of deleting", service.Namespace, service.Name, attempt, securityGroupID)
					ignore = true
				}
			}
			if !ignore {
				return fmt.Errorf("error while deleting ingress security group (%s): %v", securityGroupID, err)
			}

			attempt++
			if attempt >= 10 {
				return fmt.Errorf("error while deleting ingress security group (%s). Please file a bug report with voyager operator logs here: https://github.com/appscode/voyager/issues/new", securityGroupID)
			}
			time.Sleep(3 * time.Second)
			continue
		}
	}
	return nil
}

// Returns the instance with the specified ID
func (c *Cloud) getInstanceByID(instanceID string) (*ec2.Instance, error) {
	instances, err := c.getInstancesByIDs([]*string{&instanceID})
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, cloudprovider.InstanceNotFound
	}
	if len(instances) > 1 {
		return nil, fmt.Errorf("multiple instances found for instance: %s", instanceID)
	}

	return instances[instanceID], nil
}

func (c *Cloud) getInstancesByIDs(instanceIDs []*string) (map[string]*ec2.Instance, error) {
	instancesByID := make(map[string]*ec2.Instance)
	if len(instanceIDs) == 0 {
		return instancesByID, nil
	}

	request := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	}

	instances, err := c.ec2.DescribeInstances(request)
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		instanceID := orEmpty(instance.InstanceId)
		if instanceID == "" {
			continue
		}

		instancesByID[instanceID] = instance
	}

	return instancesByID, nil
}

// Fetches and caches instances by node names; returns an error if any cannot be found.
// This is implemented with a multi value filter on the node names, fetching the desired instances with a single query.
// TODO(therc): make all the caching more rational during the 1.4 timeframe
func (c *Cloud) getInstancesByNodeNamesCached(nodeNames sets.String) ([]*ec2.Instance, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if nodeNames.Equal(c.lastNodeNames) {
		if len(c.lastInstancesByNodeNames) > 0 {
			// We assume that if the list of nodes is the same, the underlying
			// instances have not changed. Later we might guard this with TTLs.
			glog.V(2).Infof("Returning cached instances for %v", nodeNames)
			return c.lastInstancesByNodeNames, nil
		}
	}
	names := aws.StringSlice(nodeNames.List())

	nodeNameFilter := &ec2.Filter{
		Name:   aws.String("private-dns-name"),
		Values: names,
	}

	filters := []*ec2.Filter{
		nodeNameFilter,
		newEc2Filter("instance-state-name", "running"),
	}

	filters = c.addFilters(filters)
	request := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	instances, err := c.ec2.DescribeInstances(request)
	if err != nil {
		glog.V(2).Infof("Failed to describe instances %v", nodeNames)
		return nil, err
	}

	if len(instances) == 0 {
		glog.V(3).Infof("Failed to find any instances %v", nodeNames)
		return nil, nil
	}

	glog.V(2).Infof("Caching instances for %v", nodeNames)
	c.lastNodeNames = nodeNames
	c.lastInstancesByNodeNames = instances
	return instances, nil
}

// mapInstanceToNodeName maps a EC2 instance to a k8s NodeName, by extracting the PrivateDNSName
func mapInstanceToNodeName(i *ec2.Instance) types.NodeName {
	return types.NodeName(aws.StringValue(i.PrivateDnsName))
}

// Add additional filters, to match on our tags
// This lets us run multiple k8s clusters in a single EC2 AZ
func (c *Cloud) addFilters(filters []*ec2.Filter) []*ec2.Filter {
	for k, v := range c.filterTags {
		filters = append(filters, newEc2Filter("tag:"+k, v))
	}
	if len(filters) == 0 {
		// We can't pass a zero-length Filters to AWS (it's an error)
		// So if we end up with no filters; just return nil
		return nil
	}

	return filters
}

// Returns the cluster name or an empty string
func (c *Cloud) getClusterName() string {
	return c.filterTags[TagNameKubernetesCluster]
}
