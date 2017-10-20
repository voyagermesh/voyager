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

package gce

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/compute/metadata"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"gopkg.in/gcfg.v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/flowcontrol"
	netsets "k8s.io/kubernetes/pkg/util/net/sets"
)

const (
	ProviderName = "gce"

	operationPollInterval        = 3 * time.Second
	operationPollTimeoutDuration = 30 * time.Minute

	// Each page can have 500 results, but we cap how many pages
	// are iterated through to prevent infinite loops if the API
	// were to continuously return a nextPageToken.
	maxPages = 25
)

// GCECloud is an implementation of Interface, LoadBalancer and Instances for Google Compute Engine.
type GCECloud struct {
	service                  *compute.Service
	containerService         *container.Service
	projectID                string
	region                   string
	localZone                string   // The zone in which we are running
	managedZones             []string // List of zones we are spanning (for multi-AZ clusters, primarily when running on master)
	networkURL               string
	nodeTags                 []string // List of tags to use on firewall rules for load balancers
	nodeInstancePrefix       string   // If non-"", an advisory prefix for all nodes in the cluster
	useMetadataServer        bool
	operationPollRateLimiter flowcontrol.RateLimiter
}

type Config struct {
	Global struct {
		TokenURL           string   `gcfg:"token-url"`
		TokenBody          string   `gcfg:"token-body"`
		ProjectID          string   `gcfg:"project-id"`
		NetworkName        string   `gcfg:"network-name"`
		NodeTags           []string `gcfg:"node-tags"`
		NodeInstancePrefix string   `gcfg:"node-instance-prefix"`
		Multizone          bool     `gcfg:"multizone"`
	}
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) { return newGCECloud(config) })
}

// Raw access to the underlying GCE service, probably should only be used for e2e tests
func (gce *GCECloud) GetComputeService() *compute.Service {
	return gce.service
}

func getProjectAndZone() (string, string, error) {
	result, err := metadata.Get("instance/zone")
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(result, "/")
	if len(parts) != 4 {
		return "", "", fmt.Errorf("unexpected response: %s", result)
	}
	zone := parts[3]
	projectID, err := metadata.ProjectID()
	if err != nil {
		return "", "", err
	}
	return projectID, zone, nil
}

func getNetworkNameViaMetadata() (string, error) {
	result, err := metadata.Get("instance/network-interfaces/0/network")
	if err != nil {
		return "", err
	}
	parts := strings.Split(result, "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected response: %s", result)
	}
	return parts[3], nil
}

func getNetworkNameViaAPICall(svc *compute.Service, projectID string) (string, error) {
	// TODO: use PageToken to list all not just the first 500
	networkList, err := svc.Networks.List(projectID).Do()
	if err != nil {
		return "", err
	}

	if networkList == nil || len(networkList.Items) <= 0 {
		return "", fmt.Errorf("GCE Network List call returned no networks for project %q.", projectID)
	}

	return networkList.Items[0].Name, nil
}

func getZonesForRegion(svc *compute.Service, projectID, region string) ([]string, error) {
	// TODO: use PageToken to list all not just the first 500
	listCall := svc.Zones.List(projectID)

	// Filtering by region doesn't seem to work
	// (tested in https://cloud.google.com/compute/docs/reference/latest/zones/list)
	// listCall = listCall.Filter("region eq " + region)

	res, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("unexpected response listing zones: %v", err)
	}
	zones := []string{}
	for _, zone := range res.Items {
		regionName := lastComponent(zone.Region)
		if regionName == region {
			zones = append(zones, zone.Name)
		}
	}
	return zones, nil
}

// newGCECloud creates a new instance of GCECloud.
func newGCECloud(config io.Reader) (*GCECloud, error) {
	projectID, zone, err := getProjectAndZone()
	if err != nil {
		return nil, err
	}

	region, err := GetGCERegion(zone)
	if err != nil {
		return nil, err
	}

	networkName, err := getNetworkNameViaMetadata()
	if err != nil {
		return nil, err
	}
	networkURL := gceNetworkURL(projectID, networkName)

	// By default, Kubernetes clusters only run against one zone
	managedZones := []string{zone}

	tokenSource := google.ComputeTokenSource("")
	var nodeTags []string
	var nodeInstancePrefix string
	if config != nil {
		var cfg Config
		if err := gcfg.ReadInto(&cfg, config); err != nil {
			glog.Errorf("Couldn't read config: %v", err)
			return nil, err
		}
		glog.Infof("Using GCE provider config %+v", cfg)
		if cfg.Global.ProjectID != "" {
			projectID = cfg.Global.ProjectID
		}
		if cfg.Global.NetworkName != "" {
			if strings.Contains(cfg.Global.NetworkName, "/") {
				networkURL = cfg.Global.NetworkName
			} else {
				networkURL = gceNetworkURL(cfg.Global.ProjectID, cfg.Global.NetworkName)
			}
		}
		if cfg.Global.TokenURL != "" {
			tokenSource = NewAltTokenSource(cfg.Global.TokenURL, cfg.Global.TokenBody)
		}
		nodeTags = cfg.Global.NodeTags
		nodeInstancePrefix = cfg.Global.NodeInstancePrefix
		if cfg.Global.Multizone {
			managedZones = nil // Use all zones in region
		}
	}

	return CreateGCECloud(projectID, region, zone, managedZones, networkURL, nodeTags, nodeInstancePrefix, tokenSource, true /* useMetadataServer */)
}

// Creates a GCECloud object using the specified parameters.
// If no networkUrl is specified, loads networkName via rest call.
// If no tokenSource is specified, uses oauth2.DefaultTokenSource.
// If managedZones is nil / empty all zones in the region will be managed.
func CreateGCECloud(projectID, region, zone string, managedZones []string, networkURL string, nodeTags []string, nodeInstancePrefix string, tokenSource oauth2.TokenSource, useMetadataServer bool) (*GCECloud, error) {
	if tokenSource == nil {
		var err error
		tokenSource, err = google.DefaultTokenSource(
			oauth2.NoContext,
			compute.CloudPlatformScope,
			compute.ComputeScope)
		glog.Infof("Using DefaultTokenSource %#v", tokenSource)
		if err != nil {
			return nil, err
		}
	} else {
		glog.Infof("Using existing Token Source %#v", tokenSource)
	}

	if err := wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
		if _, err := tokenSource.Token(); err != nil {
			glog.Errorf("error fetching initial token: %v", err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	client := oauth2.NewClient(oauth2.NoContext, tokenSource)
	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	containerSvc, err := container.New(client)
	if err != nil {
		return nil, err
	}

	if networkURL == "" {
		networkName, err := getNetworkNameViaAPICall(svc, projectID)
		if err != nil {
			return nil, err
		}
		networkURL = gceNetworkURL(projectID, networkName)
	}

	if len(managedZones) == 0 {
		managedZones, err = getZonesForRegion(svc, projectID, region)
		if err != nil {
			return nil, err
		}
	}
	if len(managedZones) != 1 {
		glog.Infof("managing multiple zones: %v", managedZones)
	}

	operationPollRateLimiter := flowcontrol.NewTokenBucketRateLimiter(10, 100) // 10 qps, 100 bucket size.

	return &GCECloud{
		service:                  svc,
		containerService:         containerSvc,
		projectID:                projectID,
		region:                   region,
		localZone:                zone,
		managedZones:             managedZones,
		networkURL:               networkURL,
		nodeTags:                 nodeTags,
		nodeInstancePrefix:       nodeInstancePrefix,
		useMetadataServer:        useMetadataServer,
		operationPollRateLimiter: operationPollRateLimiter,
	}, nil
}

// ProviderName returns the cloud provider ID.
func (gce *GCECloud) ProviderName() string {
	return ProviderName
}

// Firewall returns an implementation of Firewall for Google Compute Engine.
func (gce *GCECloud) Firewall() (cloudprovider.Firewall, bool) {
	return gce, true
}

func (gce *GCECloud) waitForOp(op *compute.Operation, getOperation func(operationName string) (*compute.Operation, error)) error {
	if op == nil {
		return fmt.Errorf("operation must not be nil")
	}

	if opIsDone(op) {
		return getErrorFromOp(op)
	}

	opStart := time.Now()
	opName := op.Name
	return wait.Poll(operationPollInterval, operationPollTimeoutDuration, func() (bool, error) {
		start := time.Now()
		gce.operationPollRateLimiter.Accept()
		duration := time.Now().Sub(start)
		if duration > 5*time.Second {
			glog.Infof("pollOperation: throttled %v for %v", duration, opName)
		}
		pollOp, err := getOperation(opName)
		if err != nil {
			glog.Warningf("GCE poll operation %s failed: pollOp: [%v] err: [%v] getErrorFromOp: [%v]", opName, pollOp, err, getErrorFromOp(pollOp))
		}
		done := opIsDone(pollOp)
		if done {
			duration := time.Now().Sub(opStart)
			if duration > 1*time.Minute {
				// Log the JSON. It's cleaner than the %v structure.
				enc, err := pollOp.MarshalJSON()
				if err != nil {
					glog.Warningf("waitForOperation: long operation (%v): %v (failed to encode to JSON: %v)", duration, pollOp, err)
				} else {
					glog.Infof("waitForOperation: long operation (%v): %v", duration, string(enc))
				}
			}
		}
		return done, getErrorFromOp(pollOp)
	})
}

func opIsDone(op *compute.Operation) bool {
	return op != nil && op.Status == "DONE"
}

func getErrorFromOp(op *compute.Operation) error {
	if op != nil && op.Error != nil && len(op.Error.Errors) > 0 {
		err := &googleapi.Error{
			Code:    int(op.HttpErrorStatusCode),
			Message: op.Error.Errors[0].Message,
		}
		glog.Errorf("GCE operation failed: %v", err)
		return err
	}

	return nil
}

func (gce *GCECloud) waitForGlobalOp(op *compute.Operation) error {
	return gce.waitForOp(op, func(operationName string) (*compute.Operation, error) {
		return gce.service.GlobalOperations.Get(gce.projectID, operationName).Do()
	})
}

func (gce *GCECloud) waitForRegionOp(op *compute.Operation, region string) error {
	return gce.waitForOp(op, func(operationName string) (*compute.Operation, error) {
		return gce.service.RegionOperations.Get(gce.projectID, region, operationName).Do()
	})
}

func (gce *GCECloud) waitForZoneOp(op *compute.Operation, zone string) error {
	return gce.waitForOp(op, func(operationName string) (*compute.Operation, error) {
		return gce.service.ZoneOperations.Get(gce.projectID, zone, operationName).Do()
	})
}

func isHTTPErrorCode(err error, code int) bool {
	apiErr, ok := err.(*googleapi.Error)
	return ok && apiErr.Code == code
}

func (gce *GCECloud) GetSecurityGroupName(service *apiv1.Service) string {
	//GCE requires that the name of a load balancer starts with a lower case letter.
	ret := "k8s-" + strings.ToLower(service.Name+"-"+service.Namespace)
	// Values must match the following regular expression: '[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?'
	ret = strings.TrimFunc(ret, func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLower(r) && r != '-'
	})
	if len(ret) > 64 {
		ret = ret[:64]
	}
	return ret
}

// EnsureFirewall is an implementation of LoadBalancer.EnsureLoadBalancer.
// Our load balancers in GCE consist of four separate GCE resources - a static
// IP address, a firewall rule, a target pool, and a forwarding rule. This
// function has to manage all of them.
// Due to an interesting series of design decisions, this handles both creating
// new load balancers and updating existing load balancers, recognizing when
// each is needed.
func (gce *GCECloud) EnsureFirewall(apiService *apiv1.Service, hostnames []string) error {
	hostSet := sets.NewString(hostnames...)
	hosts, err := gce.getInstancesByNames(hostSet.List())
	if err != nil {
		return err
	}

	fwName := gce.GetSecurityGroupName(apiService)
	ports := make([]apiv1.ServicePort, 0, len(apiService.Spec.Ports))
	portStr := make([]string, 0, len(apiService.Spec.Ports))
	if apiService.Spec.Type == apiv1.ServiceTypeNodePort {
		for _, p := range apiService.Spec.Ports {
			if p.NodePort == 0 {
				return fmt.Errorf("service %s/%s port %d has no associated NodePort", apiService.Namespace, apiService.Name, p.Port)
			}
			ports = append(ports, apiv1.ServicePort{
				Name:     p.Name,
				Protocol: p.Protocol,
				Port:     p.NodePort,
			})
			portStr = append(portStr, fmt.Sprintf("%s/%d", p.Protocol, p.NodePort))
		}
	} else {
		ports = apiService.Spec.Ports
		for _, p := range apiService.Spec.Ports {
			portStr = append(portStr, fmt.Sprintf("%s/%d", p.Protocol, p.Port))
		}
	}

	serviceName := types.NamespacedName{Namespace: apiService.Namespace, Name: apiService.Name}
	glog.V(2).Infof("EnsureFirewall(%v, %v, %v, %v, %v)", fwName, gce.region, portStr, hosts, serviceName)

	// Deal with the firewall next. The reason we do this here rather than last
	// is because the forwarding rule is used as the indicator that the load
	// balancer is fully created - it's what getLoadBalancer checks for.
	// Check if user specified the allow source range
	sourceRanges, err := cloudprovider.GetLoadBalancerSourceRanges(apiService)
	if err != nil {
		return err
	}
	glog.V(6).Infof("EnsureFirewall = %v, sourceRanges = %v", fwName, sourceRanges)

	firewallExists, firewallNeedsUpdate, err := gce.firewallNeedsUpdate(fwName, serviceName.String(), gce.region, "", apiService, sourceRanges)
	if err != nil {
		return err
	}
	glog.V(6).Infof("EnsureFirewall = %v, firewallExists = %v, firewallNeedsUpdate = %v", fwName, firewallExists, firewallNeedsUpdate)

	if firewallNeedsUpdate {
		desc := makeFirewallDescription(serviceName.String(), "")
		glog.V(6).Infof("EnsureFirewall = %v, desc = %v", fwName, desc)

		// Unlike forwarding rules and target pools, firewalls can be updated
		// without needing to be deleted and recreated.
		if firewallExists {
			if err := gce.updateFirewall(fwName, gce.region, desc, sourceRanges, ports, hosts); err != nil {
				return err
			}
			glog.V(4).Infof("EnsureFirewall(%v(%v)): updated firewall", fwName, serviceName)
		} else {
			if err := gce.createFirewall(fwName, gce.region, desc, sourceRanges, ports, hosts); err != nil {
				return err
			}
			glog.V(4).Infof("EnsureFirewall(%v(%v)): created firewall", fwName, serviceName)
		}
	}

	return nil
}

func (gce *GCECloud) firewallNeedsUpdate(name, serviceName, region, ipAddress string, svc *apiv1.Service, sourceRanges netsets.IPNet) (exists bool, needsUpdate bool, err error) {
	fw, err := gce.service.Firewalls.Get(gce.projectID, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return false, true, nil
		}
		return false, false, fmt.Errorf("error getting load balancer's target pool: %v", err)
	}
	if fw.Description != makeFirewallDescription(serviceName, ipAddress) {
		return true, true, nil
	}
	if len(fw.Allowed) != 1 || (fw.Allowed[0].IPProtocol != "tcp" && fw.Allowed[0].IPProtocol != "udp") {
		return true, true, nil
	}
	// Make sure the allowed ports match.
	allowedPorts := make([]string, len(svc.Spec.Ports))
	for ix := range svc.Spec.Ports {
		port := svc.Spec.Ports[ix]
		if svc.Spec.Type == apiv1.ServiceTypeNodePort {
			if port.NodePort == 0 {
				glog.Errorf("Ignoring port without NodePort defined: %v", port)
				continue
			}
			allowedPorts[ix] = strconv.Itoa(int(port.NodePort))
		} else {
			allowedPorts[ix] = strconv.Itoa(int(port.Port))
		}
	}
	if !slicesEqual(allowedPorts, fw.Allowed[0].Ports) {
		return true, true, nil
	}
	// The service controller already verified that the protocol matches on all ports, no need to check.

	actualSourceRanges, err := netsets.ParseIPNets(fw.SourceRanges...)
	if err != nil {
		// This really shouldn't happen... GCE has returned something unexpected
		glog.Warningf("Error parsing firewall SourceRanges: %v", fw.SourceRanges)
		// We don't return the error, because we can hopefully recover from this by reconfiguring the firewall
		return true, true, nil
	}

	if !sourceRanges.Equal(actualSourceRanges) {
		return true, true, nil
	}
	return true, false, nil
}

func makeFirewallDescription(serviceName, ipAddress string) string {
	if ipAddress == "" {
		return fmt.Sprintf(`{"kubernetes.io/service-name":"%s"}`, serviceName)
	}
	return fmt.Sprintf(`{"kubernetes.io/service-name":"%s", "kubernetes.io/service-ip":"%s"}`,
		serviceName, ipAddress)
}

func slicesEqual(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func (gce *GCECloud) createFirewall(fwName, region, desc string, sourceRanges netsets.IPNet, ports []apiv1.ServicePort, hosts []*gceInstance) error {
	firewall, err := gce.firewallObject(fwName, region, desc, sourceRanges, ports, hosts)
	if err != nil {
		return err
	}
	op, err := gce.service.Firewalls.Insert(gce.projectID, firewall).Do()
	if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
		return err
	}
	if op != nil {
		err = gce.waitForGlobalOp(op)
		if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
			return err
		}
	}
	return nil
}

func (gce *GCECloud) updateFirewall(fwName, region, desc string, sourceRanges netsets.IPNet, ports []apiv1.ServicePort, hosts []*gceInstance) error {
	firewall, err := gce.firewallObject(fwName, region, desc, sourceRanges, ports, hosts)
	if err != nil {
		return err
	}
	op, err := gce.service.Firewalls.Update(gce.projectID, fwName, firewall).Do()
	if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
		return err
	}
	if op != nil {
		err = gce.waitForGlobalOp(op)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gce *GCECloud) firewallObject(fwName, region, desc string, sourceRanges netsets.IPNet, ports []apiv1.ServicePort, hosts []*gceInstance) (*compute.Firewall, error) {
	allowedPorts := make([]string, len(ports))
	for ix := range ports {
		allowedPorts[ix] = strconv.Itoa(int(ports[ix].Port))
	}

	// If the node tags to be used for this cluster have been predefined in the
	// provider config, just use them. Otherwise, invoke computeHostTags method to get the tags.
	hostTags := gce.nodeTags
	if len(hostTags) == 0 {
		var err error
		if hostTags, err = gce.computeHostTags(hosts); err != nil {
			return nil, fmt.Errorf("No node tags supplied and also failed to parse the given lists of hosts for tags. Abort creating firewall rule. Reason: %v.", err)
		}
	}

	firewall := &compute.Firewall{
		Name:         fwName,
		Description:  desc,
		Network:      gce.networkURL,
		SourceRanges: sourceRanges.StringSlice(),
		TargetTags:   hostTags,
		Allowed: []*compute.FirewallAllowed{
			{
				// TODO: Make this more generic. Currently this method is only
				// used to create firewall rules for loadbalancers, which have
				// exactly one protocol, so we can never end up with a list of
				// mixed TCP and UDP ports. It should be possible to use a
				// single firewall rule for both a TCP and UDP lb.
				IPProtocol: strings.ToLower(string(ports[0].Protocol)),
				Ports:      allowedPorts,
			},
		},
	}
	return firewall, nil
}

// ComputeHostTags grabs all tags from all instances being added to the pool.
// * The longest tag that is a prefix of the instance name is used
// * If any instance has no matching prefix tag, return error
// Invoking this method to get host tags is risky since it depends on the format
// of the host names in the cluster. Only use it as a fallback if gce.nodeTags
// is unspecified
func (gce *GCECloud) computeHostTags(hosts []*gceInstance) ([]string, error) {
	// TODO: We could store the tags in gceInstance, so we could have already fetched it
	hostNamesByZone := make(map[string]map[string]bool) // map of zones -> map of names -> bool (for easy lookup)
	nodeInstancePrefix := gce.nodeInstancePrefix
	for _, host := range hosts {
		if !strings.HasPrefix(host.Name, gce.nodeInstancePrefix) {
			glog.Warningf("instance '%s' does not conform to prefix '%s', ignoring filter", host, gce.nodeInstancePrefix)
			nodeInstancePrefix = ""
		}

		z, ok := hostNamesByZone[host.Zone]
		if !ok {
			z = make(map[string]bool)
			hostNamesByZone[host.Zone] = z
		}
		z[host.Name] = true
	}

	tags := sets.NewString()

	for zone, hostNames := range hostNamesByZone {
		pageToken := ""
		page := 0
		for ; page == 0 || (pageToken != "" && page < maxPages); page++ {
			listCall := gce.service.Instances.List(gce.projectID, zone)

			if nodeInstancePrefix != "" {
				// Add the filter for hosts
				listCall = listCall.Filter("name eq " + nodeInstancePrefix + ".*")
			}

			// Add the fields we want
			// TODO(zmerlynn): Internal bug 29524655
			// listCall = listCall.Fields("items(name,tags)")

			if pageToken != "" {
				listCall = listCall.PageToken(pageToken)
			}

			res, err := listCall.Do()
			if err != nil {
				return nil, err
			}
			pageToken = res.NextPageToken
			for _, instance := range res.Items {
				if !hostNames[instance.Name] {
					continue
				}

				longest_tag := ""
				for _, tag := range instance.Tags.Items {
					if instance.Name != tag && len(tag) > len(longest_tag) {
						longest_tag = tag
					}
				}
				if len(longest_tag) > 0 {
					tags.Insert(longest_tag)
				} else {
					return nil, fmt.Errorf("Could not find any tag that is a prefix of instance name for instance %s", instance.Name)
				}
			}
		}
		if page >= maxPages {
			glog.Errorf("computeHostTags exceeded maxPages=%d for Instances.List: truncating.", maxPages)
		}
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("No instances found")
	}
	return tags.List(), nil
}

// EnsureFirewallDeleted is an implementation of Firewall.EnsureFirewallDeleted.
func (gce *GCECloud) EnsureFirewallDeleted(service *apiv1.Service) error {
	fwName := gce.GetSecurityGroupName(service)
	glog.V(2).Infof("EnsureFirewallDeleted(%v, %v, %v, %v)", service.Namespace, service.Name, fwName,
		gce.region)

	errs := utilerrors.AggregateGoroutines(
		func() error { return gce.deleteFirewall(fwName, gce.region) },
	)
	if errs != nil {
		return utilerrors.Flatten(errs)
	}
	return nil
}

func (gce *GCECloud) deleteFirewall(fwName, region string) error {
	op, err := gce.service.Firewalls.Delete(gce.projectID, fwName).Do()
	if err != nil && isHTTPErrorCode(err, http.StatusNotFound) {
		glog.Infof("Firewall %s already deleted. Continuing to delete other resources.", fwName)
	} else if err != nil {
		glog.Warningf("Failed to delete firewall %s, got error %v", fwName, err)
		return err
	} else {
		if err := gce.waitForGlobalOp(op); err != nil {
			glog.Warningf("Failed waiting for Firewall %s to be deleted.  Got error: %v", fwName, err)
			return err
		}
	}
	return nil
}

// Take a GCE instance 'hostname' and break it down to something that can be fed
// to the GCE API client library.  Basically this means reducing 'kubernetes-
// node-2.c.my-proj.internal' to 'kubernetes-node-2' if necessary.
func canonicalizeInstanceName(name string) string {
	ix := strings.Index(name, ".")
	if ix != -1 {
		name = name[:ix]
	}
	return name
}

func gceNetworkURL(project, network string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s", project, network)
}

// GetGCERegion returns region of the gce zone. Zone names
// are of the form: ${region-name}-${ix}.
// For example, "us-central1-b" has a region of "us-central1".
// So we look for the last '-' and trim to just before that.
func GetGCERegion(zone string) (string, error) {
	ix := strings.LastIndex(zone, "-")
	if ix == -1 {
		return "", fmt.Errorf("unexpected zone: %s", zone)
	}
	return zone[:ix], nil
}

type gceInstance struct {
	Zone  string
	Name  string
	ID    uint64
	Disks []*compute.AttachedDisk
	Type  string
}

type gceDisk struct {
	Zone string
	Name string
	Kind string
}

// Gets the named instances, returning cloudprovider.InstanceNotFound if any instance is not found
func (gce *GCECloud) getInstancesByNames(names []string) ([]*gceInstance, error) {
	instances := make(map[string]*gceInstance)
	remaining := len(names)

	nodeInstancePrefix := gce.nodeInstancePrefix
	for _, name := range names {
		name = canonicalizeInstanceName(name)
		if !strings.HasPrefix(name, gce.nodeInstancePrefix) {
			glog.Warningf("instance '%s' does not conform to prefix '%s', removing filter", name, gce.nodeInstancePrefix)
			nodeInstancePrefix = ""
		}
		instances[name] = nil
	}

	for _, zone := range gce.managedZones {
		if remaining == 0 {
			break
		}

		pageToken := ""
		page := 0
		for ; page == 0 || (pageToken != "" && page < maxPages); page++ {
			listCall := gce.service.Instances.List(gce.projectID, zone)

			if nodeInstancePrefix != "" {
				// Add the filter for hosts
				listCall = listCall.Filter("name eq " + nodeInstancePrefix + ".*")
			}

			// TODO(zmerlynn): Internal bug 29524655
			// listCall = listCall.Fields("items(name,id,disks,machineType)")
			if pageToken != "" {
				listCall.PageToken(pageToken)
			}

			res, err := listCall.Do()
			if err != nil {
				return nil, err
			}
			pageToken = res.NextPageToken
			for _, i := range res.Items {
				name := i.Name
				if _, ok := instances[name]; !ok {
					continue
				}

				instance := &gceInstance{
					Zone:  zone,
					Name:  name,
					ID:    i.Id,
					Disks: i.Disks,
					Type:  lastComponent(i.MachineType),
				}
				instances[name] = instance
				remaining--
			}
		}
		if page >= maxPages {
			glog.Errorf("getInstancesByNames exceeded maxPages=%d for Instances.List: truncating.", maxPages)
		}
	}

	instanceArray := make([]*gceInstance, len(names))
	for i, name := range names {
		name = canonicalizeInstanceName(name)
		instance := instances[name]
		if instance == nil {
			glog.Errorf("Failed to retrieve instance: %q", name)
			return nil, cloudprovider.InstanceNotFound
		}
		instanceArray[i] = instances[name]
	}

	return instanceArray, nil
}

// Returns the last component of a URL, i.e. anything after the last slash
// If there is no slash, returns the whole string
func lastComponent(s string) string {
	lastSlash := strings.LastIndex(s, "/")
	if lastSlash != -1 {
		s = s[lastSlash+1:]
	}
	return s
}
