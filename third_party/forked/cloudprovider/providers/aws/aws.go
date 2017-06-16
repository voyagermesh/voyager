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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/golang/glog"
	"gopkg.in/gcfg.v1"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/service"
	aws_credentials "k8s.io/kubernetes/pkg/credentialprovider/aws"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util/sets"
)

// ProviderName is the name of this cloud provider.
const ProviderName = "aws"

// TagNameKubernetesCluster is the tag name we use to differentiate multiple
// logically independent clusters running in the same AZ
const TagNameKubernetesCluster = "KubernetesCluster"

// TagNameKubernetesService is the tag name we use to differentiate multiple
// services. Used currently for ELBs only.
const TagNameKubernetesService = "kubernetes.io/service-name"

// TagNameSubnetInternalELB is the tag name used on a subnet to designate that
// it should be used for internal ELBs
const TagNameSubnetInternalELB = "kubernetes.io/role/internal-elb"

// TagNameSubnetPublicELB is the tag name used on a subnet to designate that
// it should be used for internet ELBs
const TagNameSubnetPublicELB = "kubernetes.io/role/elb"

// ServiceAnnotationLoadBalancerInternal is the annotation used on the service
// to indicate that we want an internal ELB.
// Currently we accept only the value "0.0.0.0/0" - other values are an error.
// This lets us define more advanced semantics in future.
const ServiceAnnotationLoadBalancerInternal = "service.beta.kubernetes.io/aws-load-balancer-internal"

// ServiceAnnotationLoadBalancerProxyProtocol is the annotation used on the
// service to enable the proxy protocol on an ELB. Right now we only accept the
// value "*" which means enable the proxy protocol on all ELB backends. In the
// future we could adjust this to allow setting the proxy protocol only on
// certain backends.
const ServiceAnnotationLoadBalancerProxyProtocol = "service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"

// ServiceAnnotationLoadBalancerAccessLogEmitInterval is the annotation used to
// specify access log emit interval.
const ServiceAnnotationLoadBalancerAccessLogEmitInterval = "service.beta.kubernetes.io/aws-load-balancer-access-log-emit-interval"

// ServiceAnnotationLoadBalancerAccessLogEnabled is the annotation used on the
// service to enable or disable access logs.
const ServiceAnnotationLoadBalancerAccessLogEnabled = "service.beta.kubernetes.io/aws-load-balancer-access-log-enabled"

// ServiceAnnotationLoadBalancerAccessLogS3BucketName is the annotation used to
// specify access log s3 bucket name.
const ServiceAnnotationLoadBalancerAccessLogS3BucketName = "service.beta.kubernetes.io/aws-load-balancer-access-log-s3-bucket-name"

// ServiceAnnotationLoadBalancerAccessLogS3BucketPrefix is the annotation used
// to specify access log s3 bucket prefix.
const ServiceAnnotationLoadBalancerAccessLogS3BucketPrefix = "service.beta.kubernetes.io/aws-load-balancer-access-log-s3-bucket-prefix"

// ServiceAnnotationLoadBalancerConnectionDrainingEnabled is the annnotation
// used on the service to enable or disable connection draining.
const ServiceAnnotationLoadBalancerConnectionDrainingEnabled = "service.beta.kubernetes.io/aws-load-balancer-connection-draining-enabled"

// ServiceAnnotationLoadBalancerConnectionDrainingTimeout is the annotation
// used on the service to specify a connection draining timeout.
const ServiceAnnotationLoadBalancerConnectionDrainingTimeout = "service.beta.kubernetes.io/aws-load-balancer-connection-draining-timeout"

// ServiceAnnotationLoadBalancerConnectionIdleTimeout is the annotation used
// on the service to specify the idle connection timeout.
const ServiceAnnotationLoadBalancerConnectionIdleTimeout = "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout"

// ServiceAnnotationLoadBalancerCrossZoneLoadBalancingEnabled is the annotation
// used on the service to enable or disable cross-zone load balancing.
const ServiceAnnotationLoadBalancerCrossZoneLoadBalancingEnabled = "service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled"

// ServiceAnnotationLoadBalancerCertificate is the annotation used on the
// service to request a secure listener. Value is a valid certificate ARN.
// For more, see http://docs.aws.amazon.com/ElasticLoadBalancing/latest/DeveloperGuide/elb-listener-config.html
// CertARN is an IAM or CM certificate ARN, e.g. arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012
const ServiceAnnotationLoadBalancerCertificate = "service.beta.kubernetes.io/aws-load-balancer-ssl-cert"

// ServiceAnnotationLoadBalancerSSLPorts is the annotation used on the service
// to specify a comma-separated list of ports that will use SSL/HTTPS
// listeners. Defaults to '*' (all).
const ServiceAnnotationLoadBalancerSSLPorts = "service.beta.kubernetes.io/aws-load-balancer-ssl-ports"

// ServiceAnnotationLoadBalancerBEProtocol is the annotation used on the service
// to specify the protocol spoken by the backend (pod) behind a listener.
// If `http` (default) or `https`, an HTTPS listener that terminates the
//  connection and parses headers is created.
// If set to `ssl` or `tcp`, a "raw" SSL listener is used.
// If set to `http` and `aws-load-balancer-ssl-cert` is not used then
// a HTTP listener is used.
const ServiceAnnotationLoadBalancerBEProtocol = "service.beta.kubernetes.io/aws-load-balancer-backend-protocol"

const (
	// volumeAttachmentStatusTimeout is the maximum time to wait for a volume attach/detach to complete
	volumeAttachmentStatusTimeout = 30 * time.Minute
	// volumeAttachmentConsecutiveErrorLimit is the number of consecutive errors we will ignore when waiting for a volume to attach/detach
	volumeAttachmentStatusConsecutiveErrorLimit = 10
	// volumeAttachmentErrorDelay is the amount of time we wait before retrying after encountering an error,
	// while waiting for a volume attach/detach to complete
	volumeAttachmentStatusErrorDelay = 20 * time.Second
	// volumeAttachmentStatusPollInterval is the interval at which we poll the volume,
	// while waiting for a volume attach/detach to complete
	volumeAttachmentStatusPollInterval = 10 * time.Second
)

// Maps from backend protocol to ELB protocol
var backendProtocolMapping = map[string]string{
	"https": "https",
	"http":  "https",
	"ssl":   "ssl",
	"tcp":   "ssl",
}

// MaxReadThenCreateRetries sets the maximum number of attempts we will make when
// we read to see if something exists and then try to create it if we didn't find it.
// This can fail once in a consistent system if done in parallel
// In an eventually consistent system, it could fail unboundedly
const MaxReadThenCreateRetries = 30

// DefaultVolumeType specifies which storage to use for newly created Volumes
// TODO: Remove when user/admin can configure volume types and thus we don't
// need hardcoded defaults.
const DefaultVolumeType = "gp2"

// DefaultMaxEBSVolumes is the limit for volumes attached to an instance.
// Amazon recommends no more than 40; the system root volume uses at least one.
// See http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/volume_limits.html#linux-specific-volume-limits
const DefaultMaxEBSVolumes = 39

// Used to call aws_credentials.Init() just once
var once sync.Once

// Services is an abstraction over AWS, to allow mocking/other implementations
type Services interface {
	Compute(region string) (EC2, error)
	LoadBalancing(region string) (ELB, error)
	Autoscaling(region string) (ASG, error)
	Metadata() (EC2Metadata, error)
}

// EC2 is an abstraction over AWS', to allow mocking/other implementations
// Note that the DescribeX functions return a list, so callers don't need to deal with paging
// TODO: Should we rename this to AWS (EBS & ELB are not technically part of EC2)
type EC2 interface {
	// Query EC2 for instances matching the filter
	DescribeInstances(request *ec2.DescribeInstancesInput) ([]*ec2.Instance, error)

	// Attach a volume to an instance
	AttachVolume(*ec2.AttachVolumeInput) (*ec2.VolumeAttachment, error)
	// Detach a volume from an instance it is attached to
	DetachVolume(request *ec2.DetachVolumeInput) (resp *ec2.VolumeAttachment, err error)
	// Lists volumes
	DescribeVolumes(request *ec2.DescribeVolumesInput) ([]*ec2.Volume, error)
	// Create an EBS volume
	CreateVolume(request *ec2.CreateVolumeInput) (resp *ec2.Volume, err error)
	// Delete an EBS volume
	DeleteVolume(*ec2.DeleteVolumeInput) (*ec2.DeleteVolumeOutput, error)

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

// ASG is a simple pass-through of the Autoscaling client interface, which
// allows for testing.
type ASG interface {
	UpdateAutoScalingGroup(*autoscaling.UpdateAutoScalingGroupInput) (*autoscaling.UpdateAutoScalingGroupOutput, error)
	DescribeAutoScalingGroups(*autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

// EC2Metadata is an abstraction over the AWS metadata service.
type EC2Metadata interface {
	// Query the EC2 metadata service (used to discover instance-id etc)
	GetMetadata(path string) (string, error)
}

// InstanceGroups is an interface for managing cloud-managed instance groups / autoscaling instance groups
// TODO: Allow other clouds to implement this
type InstanceGroups interface {
	// Set the size to the fixed size
	ResizeInstanceGroup(instanceGroupName string, size int) error
	// Queries the cloud provider for information about the specified instance group
	DescribeInstanceGroup(instanceGroupName string) (InstanceGroupInfo, error)
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
	asg      ASG
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

func (p *awsSDKProvider) Autoscaling(regionName string) (ASG, error) {
	client := autoscaling.New(session.New(&aws.Config{
		Region:      &regionName,
		Credentials: p.creds,
	}))

	p.addHandlers(regionName, &client.Handlers)

	return client, nil
}

func (p *awsSDKProvider) Metadata() (EC2Metadata, error) {
	client := ec2metadata.New(session.New(&aws.Config{}))
	return client, nil
}

// stringPointerArray creates a slice of string pointers from a slice of strings
// Deprecated: consider using aws.StringSlice - but note the slightly different behaviour with a nil input
func stringPointerArray(orig []string) []*string {
	if orig == nil {
		return nil
	}
	return aws.StringSlice(orig)
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

func getInstanceType(metadata EC2Metadata) (string, error) {
	return metadata.GetMetadata("instance-type")
}

func getAvailabilityZone(metadata EC2Metadata) (string, error) {
	return metadata.GetMetadata("placement/availability-zone")
}

func isRegionValid(region string) bool {
	for _, r := range aws_credentials.AWSRegions {
		if r == region {
			return true
		}
	}
	return false
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

	asg, err := awsServices.Autoscaling(regionName)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS autoscaling client: %v", err)
	}

	awsCloud := &Cloud{
		ec2:      ec2,
		elb:      elb,
		asg:      asg,
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

	// Register handler for ECR credentials
	once.Do(func() {
		aws_credentials.Init()
	})

	return awsCloud, nil
}

// Clusters returns the list of clusters.
func (c *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
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
func (c *Cloud) ensureClusterTags(resourceID string, tags []*ec2.Tag) error {
	actualTags := make(map[string]string)
	for _, tag := range tags {
		actualTags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}

	addTags := make(map[string]string)
	for k, expected := range c.filterTags {
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
func (c *Cloud) ensureSecurityGroup(name string, description string) (string, error) {
	groupID, err := c.ensureUntaggedSecurityGroup(name, description)
	if err != nil {
		return "", err
	}

	err = c.createTags(groupID, c.filterTags)
	if err != nil {
		// If we retry, ensureClusterTags will recover from this - it
		// will add the missing tags.  We could delete the security
		// group here, but that doesn't feel like the right thing, as
		// the caller is likely to retry the create
		return "", fmt.Errorf("error tagging security group: %v", err)
	}
	return groupID, nil
}

// Makes sure the security group exists.
// For multi-cluster isolation, name must be globally unique, for example derived from the service UUID.
// Returns the security group id or error
func (c *Cloud) ensureUntaggedSecurityGroup(name string, description string) (string, error) {
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
			err := c.ensureClusterTags(aws.StringValue(securityGroups[0].GroupId), securityGroups[0].Tags)
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

	err := c.createTags(groupID, c.filterTags)
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

// EnsureFirewall implements LoadBalancer.EnsureLoadBalancer
func (c *Cloud) EnsureFirewall(apiService *api.Service, hostname string) error {
	glog.V(2).Infof("EnsureLoadBalancer(%v, %v, %v, %v, %v, %v)",
		apiService.Namespace, apiService.Name, c.region, apiService.Spec.LoadBalancerIP, apiService.Spec.Ports, hostname)

	if apiService.Spec.SessionAffinity != api.ServiceAffinityNone {
		// ELB supports sticky sessions, but only when configured for HTTP/HTTPS
		return fmt.Errorf("unsupported load balancer affinity: %v", apiService.Spec.SessionAffinity)
	}

	if len(apiService.Spec.Ports) == 0 {
		return errors.New("requested load balancer with no ports")
	}

	for _, port := range apiService.Spec.Ports {
		if port.Protocol != api.ProtocolTCP {
			return errors.New("Only TCP LoadBalancer is supported for AWS ELB")
		}
	}

	if apiService.Spec.LoadBalancerIP != "" {
		return errors.New("LoadBalancerIP cannot be specified for AWS ELB")
	}

	hostSet := sets.NewString(hostname)
	instances, err := c.getInstancesByNodeNamesCached(hostSet)
	if err != nil {
		return err
	}

	sourceRanges, err := service.GetLoadBalancerSourceRanges(apiService)
	if err != nil {
		return err
	}

	loadBalancerName := cloudprovider.GetLoadBalancerName(apiService)
	serviceName := types.NamespacedName{Namespace: apiService.Namespace, Name: apiService.Name}

	// Create a security group for the load balancer
	var securityGroupID string
	{
		sgName := "k8s-elb-" + loadBalancerName
		sgDescription := fmt.Sprintf("Security group for Kubernetes DaemonNode %s (%v)", loadBalancerName, serviceName)
		securityGroupID, err = c.ensureUntaggedSecurityGroup(sgName, sgDescription)
		if err != nil {
			glog.Error("Error creating load balancer security group: ", err)
			return err
		}
		err = c.createTags(securityGroupID, map[string]string{
			"AppsCodeCluster": c.getClusterName(),
		})
		if err != nil {
			// If we retry, ensureClusterTags will recover from this - it
			// will add the missing tags.  We could delete the security
			// group here, but that doesn't feel like the right thing, as
			// the caller is likely to retry the create
			return fmt.Errorf("error tagging security group: %v", err)
		}

		ec2SourceRanges := []*ec2.IpRange{}
		for _, sourceRange := range sourceRanges.StringSlice() {
			ec2SourceRanges = append(ec2SourceRanges, &ec2.IpRange{CidrIp: aws.String(sourceRange)})
		}

		permissions := NewIPPermissionSet()
		for _, port := range apiService.Spec.Ports {
			portInt64 := int64(port.Port)
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

	err = c.updateInstanceSecurityGroupsForFirewall(securityGroupID, instances)
	if err != nil {
		glog.Warning("Error opening ingress rules for the load balancer to the instances: ", err)
		return err
	}

	{
		// Add to network interface
		for _, instance := range instances {
			// Get the actual list of groups that allow ingress from the load-balancer
			attrRequest := &ec2.ModifyInstanceAttributeInput{}
			attrRequest.InstanceId = instance.InstanceId
			attrRequest.Groups = []*string{aws.String(securityGroupID)}
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

	// TODO: Wait for creation?
	return nil
}

// Returns the first security group for an instance, or nil
// We only create instances with one security group, so we don't expect multiple security groups.
// However, if there are multiple security groups, we will choose the one tagged with our cluster filter.
// Otherwise we will return an error.
func findSecurityGroupForInstance(instance *ec2.Instance, taggedSecurityGroups map[string]*ec2.SecurityGroup) (*ec2.GroupIdentifier, error) {
	instanceID := aws.StringValue(instance.InstanceId)

	var tagged []*ec2.GroupIdentifier
	var untagged []*ec2.GroupIdentifier
	for _, group := range instance.SecurityGroups {
		groupID := aws.StringValue(group.GroupId)
		if groupID == "" {
			glog.Warningf("Ignoring security group without id for instance %q: %v", instanceID, group)
			continue
		}
		_, isTagged := taggedSecurityGroups[groupID]
		if isTagged {
			tagged = append(tagged, group)
		} else {
			untagged = append(untagged, group)
		}
	}

	if len(tagged) > 0 {
		// We create instances with one SG
		// If users create multiple SGs, they must tag one of them as being k8s owned
		if len(tagged) != 1 {
			return nil, fmt.Errorf("Multiple tagged security groups found for instance %s; ensure only the k8s security group is tagged", instanceID)
		}
		return tagged[0], nil
	}

	if len(untagged) > 0 {
		// For back-compat, we will allow a single untagged SG
		if len(untagged) != 1 {
			return nil, fmt.Errorf("Multiple untagged security groups found for instance %s; ensure the k8s security group is tagged", instanceID)
		}
		return untagged[0], nil
	}

	glog.Warningf("No security group found for instance %q", instanceID)
	return nil, nil
}

// Return all the security groups that are tagged as being part of our cluster
func (c *Cloud) getTaggedSecurityGroups() (map[string]*ec2.SecurityGroup, error) {
	request := &ec2.DescribeSecurityGroupsInput{}
	request.Filters = c.addFilters(nil)
	groups, err := c.ec2.DescribeSecurityGroups(request)
	if err != nil {
		return nil, fmt.Errorf("error querying security groups: %v", err)
	}

	m := make(map[string]*ec2.SecurityGroup)
	for _, group := range groups {
		id := aws.StringValue(group.GroupId)
		if id == "" {
			glog.Warningf("Ignoring group without id: %v", group)
			continue
		}
		m[id] = group
	}
	return m, nil
}

func (s *Cloud) updateInstanceSecurityGroupsForFirewall(loadBalancerSecurityGroupId string, allInstances []*ec2.Instance) error {
	// Get the actual list of groups that allow ingress from the load-balancer
	describeRequest := &ec2.DescribeSecurityGroupsInput{}
	filters := []*ec2.Filter{}
	filters = append(filters, newEc2Filter("ip-permission.group-id", loadBalancerSecurityGroupId))
	describeRequest.Filters = s.addFilters(filters)
	actualGroups, err := s.ec2.DescribeSecurityGroups(describeRequest)
	if err != nil {
		return fmt.Errorf("error querying security groups for ELB: %v", err)
	}

	taggedSecurityGroups, err := s.getTaggedSecurityGroups()
	if err != nil {
		return fmt.Errorf("error querying for tagged security groups: %v", err)
	}

	// Open the firewall from the load balancer to the instance
	// We don't actually have a trivial way to know in advance which security group the instance is in
	// (it is probably the minion security group, but we don't easily have that).
	// However, we _do_ have the list of security groups on the instance records.

	// Map containing the changes we want to make; true to add, false to remove
	instanceSecurityGroupIds := map[string]bool{}

	// Scan instances for groups we want open
	for _, instance := range allInstances {
		securityGroup, err := findSecurityGroupForInstance(instance, taggedSecurityGroups)
		if err != nil {
			return err
		}

		if securityGroup == nil {
			glog.Warning("Ignoring instance without security group: ", orEmpty(instance.InstanceId))
			continue
		}
		id := aws.StringValue(securityGroup.GroupId)
		if id == "" {
			glog.Warningf("found security group without id: %v", securityGroup)
			continue
		}

		instanceSecurityGroupIds[id] = true
	}

	// Compare to actual groups
	for _, actualGroup := range actualGroups {
		actualGroupID := aws.StringValue(actualGroup.GroupId)
		if actualGroupID == "" {
			glog.Warning("Ignoring group without ID: ", actualGroup)
			continue
		}

		adding, found := instanceSecurityGroupIds[actualGroupID]
		if found && adding {
			// We don't need to make a change; the permission is already in place
			delete(instanceSecurityGroupIds, actualGroupID)
		} else {
			// This group is not needed by allInstances; delete it
			instanceSecurityGroupIds[actualGroupID] = false
		}
	}

	for instanceSecurityGroupId, add := range instanceSecurityGroupIds {
		if add {
			glog.V(2).Infof("Adding rule for traffic from the load balancer (%s) to instances (%s)", loadBalancerSecurityGroupId, instanceSecurityGroupId)
		} else {
			glog.V(2).Infof("Removing rule for traffic from the load balancer (%s) to instance (%s)", loadBalancerSecurityGroupId, instanceSecurityGroupId)
		}
		sourceGroupId := &ec2.UserIdGroupPair{}
		sourceGroupId.GroupId = &loadBalancerSecurityGroupId

		allProtocols := "-1"

		permission := &ec2.IpPermission{}
		permission.IpProtocol = &allProtocols
		permission.UserIdGroupPairs = []*ec2.UserIdGroupPair{sourceGroupId}

		permissions := []*ec2.IpPermission{permission}

		if add {
			changed, err := s.addSecurityGroupIngress(instanceSecurityGroupId, permissions)
			if err != nil {
				return err
			}
			if !changed {
				glog.Warning("Allowing ingress was not needed; concurrent change? groupId=", instanceSecurityGroupId)
			}
		} else {
			changed, err := s.removeSecurityGroupIngress(instanceSecurityGroupId, permissions)
			if err != nil {
				return err
			}
			if !changed {
				glog.Warning("Revoking ingress was not needed; concurrent change? groupId=", instanceSecurityGroupId)
			}
		}
	}

	return nil
}

// EnsureFirewallDeleted implements Firewall.EnsureFirewallDeleted.
func (c *Cloud) EnsureFirewallDeleted(service *api.Service) error {
	loadBalancerName := cloudprovider.GetLoadBalancerName(service)
	// Collect the security groups to delete
	var securityGroupID string

	{
		// Delete the security group(s) for the load balancer
		// Note that this is annoying: the load balancer disappears from the API immediately, but it is still
		// deleting in the background.  We get a DependencyViolation until the load balancer has deleted itself

		sgName := "k8s-elb-" + loadBalancerName

		request := &ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(c.vpcID),
					},
				},
				{
					Name: aws.String("tag:KubernetesCluster"),
					Values: []*string{
						aws.String(c.cfg.Global.KubernetesClusterTag),
					},
				},
				{
					Name: aws.String("group-name"),
					Values: []*string{
						aws.String(sgName),
					},
				},
			},
		}
		securityGroups, err := c.ec2.DescribeSecurityGroups(request)
		if err != nil {
			ignore := false
			if awsError, ok := err.(awserr.Error); ok {
				if awsError.Code() == "DependencyViolation" {
					glog.V(2).Infof("Ignoring DependencyViolation while deleting load-balancer security group (%s), assuming because LB is in process of deleting", securityGroupID)
					ignore = true
				}
			}
			if !ignore {
				return fmt.Errorf("error while deleting load balancer security group (%s): %v", securityGroupID, err)
			}
		}
		if len(securityGroups) > 1 {
			return fmt.Errorf("Multiple securitygroup found when EnsureFirewallDeleted for service %v", loadBalancerName)
		}
		for _, securityGroup := range securityGroups {
			securityGroupID = *securityGroup.GroupId
			break
		}
	}

	{
		// De-authorize the load balancer security group from the instances security group
		err := c.updateInstanceSecurityGroupsForFirewall(securityGroupID, nil)
		if err != nil {
			glog.Error("Error deregistering load balancer from instance security groups: ", err)
			return err
		}
	}

	{
		request := &ec2.DeleteSecurityGroupInput{}
		request.GroupId = &securityGroupID
		_, err := c.ec2.DeleteSecurityGroup(request)
		if err != nil {
			ignore := false
			if awsError, ok := err.(awserr.Error); ok {
				if awsError.Code() == "DependencyViolation" {
					glog.V(2).Infof("Ignoring DependencyViolation while deleting load-balancer security group (%s), assuming because LB is in process of deleting", securityGroupID)
					ignore = true
				}
			}
			if !ignore {
				return fmt.Errorf("error while deleting load balancer security group (%s): %v", securityGroupID, err)
			}
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

// mapNodeNameToPrivateDNSName maps a k8s NodeName to an AWS Instance PrivateDNSName
// This is a simple string cast
func mapNodeNameToPrivateDNSName(nodeName types.NodeName) string {
	return string(nodeName)
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
