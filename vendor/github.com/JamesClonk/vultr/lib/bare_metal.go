package lib

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// BareMetalServer represents a bare metal server on Vultr.
type BareMetalServer struct {
	ID              string      `json:"SUBID"`
	Name            string      `json:"label"`
	OS              string      `json:"os"`
	RAM             string      `json:"ram"`
	Disk            string      `json:"disk"`
	MainIP          string      `json:"main_ip"`
	CPUs            int         `json:"cpu_count"`
	Location        string      `json:"location"`
	RegionID        int         `json:"DCID,string"`
	DefaultPassword string      `json:"default_password"`
	Created         string      `json:"date_created"`
	Status          string      `json:"status"`
	NetmaskV4       string      `json:"netmask_v4"`
	GatewayV4       string      `json:"gateway_v4"`
	PlanID          int         `json:"METALPLANID"`
	V6Networks      []V6Network `json:"v6_networks"`
	Tag             string      `json:"tag"`
	OSID            string      `json:"OSID"`
	AppID           string      `json:"APPID"`
}

// BareMetalServerOptions are optional parameters to be used during bare metal server creation.
type BareMetalServerOptions struct {
	Script               int
	UserData             string
	Snapshot             string
	SSHKey               string
	ReservedIP           string
	IPV6                 bool
	DontNotifyOnActivate bool
	Hostname             string
	Tag                  string
	AppID                string
}

type bareMetalServers []BareMetalServer

func (b bareMetalServers) Len() int      { return len(b) }
func (b bareMetalServers) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b bareMetalServers) Less(i, j int) bool {
	// sort order: name, ip
	if strings.ToLower(b[i].Name) < strings.ToLower(b[j].Name) {
		return true
	} else if strings.ToLower(b[i].Name) > strings.ToLower(b[j].Name) {
		return false
	}
	return b[i].MainIP < b[j].MainIP
}

// UnmarshalJSON implements json.Unmarshaller on BareMetal.
// This is needed because the Vultr API is inconsistent in it's JSON responses for bare metal servers.
// Some fields can change type, from JSON number to JSON string and vice-versa.
func (b *BareMetalServer) UnmarshalJSON(data []byte) error {
	if b == nil {
		*b = BareMetalServer{}
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	value := fmt.Sprintf("%v", fields["cpu_count"])
	if len(value) == 0 || value == "<nil>" {
		value = "0"
	}
	cpu, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	b.CPUs = int(cpu)

	value = fmt.Sprintf("%v", fields["DCID"])
	if len(value) == 0 || value == "<nil>" {
		value = "0"
	}
	region, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	b.RegionID = int(region)

	value = fmt.Sprintf("%v", fields["METALPLANID"])
	if len(value) == 0 || value == "<nil>" {
		value = "0"
	}
	plan, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	b.PlanID = int(plan)

	value = fmt.Sprintf("%v", fields["OSID"])
	if value == "<nil>" {
		value = ""
	}
	b.OSID = value

	value = fmt.Sprintf("%v", fields["APPID"])
	if value == "<nil>" || value == "0" {
		value = ""
	}
	b.AppID = value

	b.ID = fmt.Sprintf("%v", fields["SUBID"])
	b.Name = fmt.Sprintf("%v", fields["label"])
	b.OS = fmt.Sprintf("%v", fields["os"])
	b.RAM = fmt.Sprintf("%v", fields["ram"])
	b.Disk = fmt.Sprintf("%v", fields["disk"])
	b.MainIP = fmt.Sprintf("%v", fields["main_ip"])
	b.Location = fmt.Sprintf("%v", fields["location"])
	b.DefaultPassword = fmt.Sprintf("%v", fields["default_password"])
	b.Created = fmt.Sprintf("%v", fields["date_created"])
	b.Status = fmt.Sprintf("%v", fields["status"])
	b.NetmaskV4 = fmt.Sprintf("%v", fields["netmask_v4"])
	b.GatewayV4 = fmt.Sprintf("%v", fields["gateway_v4"])

	v6networks := make([]V6Network, 0)
	if networks, ok := fields["v6_networks"].([]interface{}); ok {
		for _, network := range networks {
			if network, ok := network.(map[string]interface{}); ok {
				v6network := V6Network{
					Network:     fmt.Sprintf("%v", network["v6_network"]),
					MainIP:      fmt.Sprintf("%v", network["v6_main_ip"]),
					NetworkSize: fmt.Sprintf("%v", network["v6_network_size"]),
				}
				v6networks = append(v6networks, v6network)
			}
		}
		b.V6Networks = v6networks
	}

	b.Tag = fmt.Sprintf("%v", fields["tag"])

	return nil
}

// GetBareMetalServers returns a list of current bare metal servers on the Vultr account.
func (c *Client) GetBareMetalServers() ([]BareMetalServer, error) {
	var bareMetalServerMap map[string]BareMetalServer
	if err := c.get(`baremetal/list`, &bareMetalServerMap); err != nil {
		return nil, err
	}

	var bareMetalServerList []BareMetalServer
	for _, bareMetalServer := range bareMetalServerMap {
		bareMetalServerList = append(bareMetalServerList, bareMetalServer)
	}
	sort.Sort(bareMetalServers(bareMetalServerList))
	return bareMetalServerList, nil
}

// GetBareMetalServersByTag returns a list of all bare metal servers matching by tag.
func (c *Client) GetBareMetalServersByTag(tag string) ([]BareMetalServer, error) {
	var bareMetalServerMap map[string]BareMetalServer
	if err := c.get(`baremetal/list?tag=`+tag, &bareMetalServerMap); err != nil {
		return nil, err
	}

	var bareMetalServerList []BareMetalServer
	for _, bareMetalServer := range bareMetalServerMap {
		bareMetalServerList = append(bareMetalServerList, bareMetalServer)
	}
	sort.Sort(bareMetalServers(bareMetalServerList))
	return bareMetalServerList, nil
}

// GetBareMetalServer returns the bare metal server with the given ID.
func (c *Client) GetBareMetalServer(id string) (BareMetalServer, error) {
	var b BareMetalServer
	if err := c.get(`baremetal/list?SUBID=`+id, &b); err != nil {
		return BareMetalServer{}, err
	}
	return b, nil
}

// CreateBareMetalServer creates a new bare metal server on Vultr. BareMetalServerOptions are optional settings.
func (c *Client) CreateBareMetalServer(name string, regionID, planID, osID int, options *BareMetalServerOptions) (BareMetalServer, error) {
	values := url.Values{
		"label":       {name},
		"DCID":        {fmt.Sprintf("%v", regionID)},
		"METALPLANID": {fmt.Sprintf("%v", planID)},
		"OSID":        {fmt.Sprintf("%v", osID)},
	}

	if options != nil {
		if options.Script != 0 {
			values.Add("SCRIPTID", fmt.Sprintf("%v", options.Script))
		}

		if options.UserData != "" {
			values.Add("userdata", base64.StdEncoding.EncodeToString([]byte(options.UserData)))
		}

		if options.Snapshot != "" {
			values.Add("SNAPSHOTID", options.Snapshot)
		}

		if options.SSHKey != "" {
			values.Add("SSHKEYID", options.SSHKey)
		}

		values.Add("enable_ipv6", "no")
		if options.IPV6 {
			values.Set("enable_ipv6", "yes")
		}

		values.Add("notify_activate", "yes")
		if options.DontNotifyOnActivate {
			values.Set("notify_activate", "no")
		}

		if options.Hostname != "" {
			values.Add("hostname", options.Hostname)
		}

		if options.Tag != "" {
			values.Add("tag", options.Tag)
		}

		if options.AppID != "" {
			values.Add("APPID", options.AppID)
		}
	}

	var b BareMetalServer
	if err := c.post(`baremetal/create`, values, &b); err != nil {
		return BareMetalServer{}, err
	}
	b.Name = name
	b.RegionID = regionID
	b.PlanID = planID

	return b, nil
}

// RenameBareMetalServer renames an existing bare metal server.
func (c *Client) RenameBareMetalServer(id, name string) error {
	values := url.Values{
		"SUBID": {id},
		"label": {name},
	}

	return c.post(`baremetal/label_set`, values, nil)
}

// TagBareMetalServer replaces the tag on an existing bare metal server.
func (c *Client) TagBareMetalServer(id, tag string) error {
	values := url.Values{
		"SUBID": {id},
		"tag":   {tag},
	}

	return c.post(`baremetal/tag_set`, values, nil)
}

// HaltBareMetalServer stops an existing bare metal server.
func (c *Client) HaltBareMetalServer(id string) error {
	values := url.Values{
		"SUBID": {id},
	}

	return c.post(`baremetal/halt`, values, nil)
}

// RebootBareMetalServer reboots an existing bare metal server.
func (c *Client) RebootBareMetalServer(id string) error {
	values := url.Values{
		"SUBID": {id},
	}

	return c.post(`baremetal/reboot`, values, nil)
}

// ReinstallBareMetalServer reinstalls the operating system on an existing bare metal server.
func (c *Client) ReinstallBareMetalServer(id string) error {
	values := url.Values{
		"SUBID": {id},
	}

	return c.post(`baremetal/reinstall`, values, nil)
}

// ChangeOSofBareMetalServer changes the bare metal server to a different operating system.
func (c *Client) ChangeOSofBareMetalServer(id string, osID int) error {
	values := url.Values{
		"SUBID": {id},
		"OSID":  {fmt.Sprintf("%v", osID)},
	}

	return c.post(`baremetal/os_change`, values, nil)
}

// ListOSforBareMetalServer lists all available operating systems to which an existing bare metal server can be changed.
func (c *Client) ListOSforBareMetalServer(id string) ([]OS, error) {
	var osMap map[string]OS
	if err := c.get(`baremetal/os_change_list?SUBID=`+id, &osMap); err != nil {
		return nil, err
	}

	var os []OS
	for _, o := range osMap {
		os = append(os, o)
	}
	sort.Sort(oses(os))
	return os, nil
}

// DeleteBareMetalServer deletes an existing bare metal server.
func (c *Client) DeleteBareMetalServer(id string) error {
	values := url.Values{
		"SUBID": {id},
	}

	return c.post(`baremetal/destroy`, values, nil)
}

// BandwidthOfBareMetalServer retrieves the bandwidth used by a bare metal server.
func (c *Client) BandwidthOfBareMetalServer(id string) ([]map[string]string, error) {
	var bandwidthMap map[string][][]interface{}
	if err := c.get(`server/bandwidth?SUBID=`+id, &bandwidthMap); err != nil {
		return nil, err
	}

	var bandwidth []map[string]string
	// parse incoming bytes
	for _, b := range bandwidthMap["incoming_bytes"] {
		bMap := make(map[string]string)
		bMap["date"] = fmt.Sprintf("%v", b[0])
		var bytes int64
		switch b[1].(type) {
		case float64:
			bytes = int64(b[1].(float64))
		case int64:
			bytes = b[1].(int64)
		}
		bMap["incoming"] = fmt.Sprintf("%v", bytes)
		bandwidth = append(bandwidth, bMap)
	}

	// parse outgoing bytes (we'll assume that incoming and outgoing dates are always a match)
	for _, b := range bandwidthMap["outgoing_bytes"] {
		for i := range bandwidth {
			if bandwidth[i]["date"] == fmt.Sprintf("%v", b[0]) {
				var bytes int64
				switch b[1].(type) {
				case float64:
					bytes = int64(b[1].(float64))
				case int64:
					bytes = b[1].(int64)
				}
				bandwidth[i]["outgoing"] = fmt.Sprintf("%v", bytes)
				break
			}
		}
	}

	return bandwidth, nil
}

// ChangeApplicationofBareMetalServer changes the bare metal server to a different application.
func (c *Client) ChangeApplicationofBareMetalServer(id string, appID string) error {
	values := url.Values{
		"SUBID": {id},
		"APPID": {appID},
	}

	return c.post(`baremetal/app_change`, values, nil)
}

// ListApplicationsforBareMetalServer lists all available operating systems to which an existing bare metal server can be changed.
func (c *Client) ListApplicationsforBareMetalServer(id string) ([]Application, error) {
	var appMap map[string]Application
	if err := c.get(`baremetal/app_change_list?SUBID=`+id, &appMap); err != nil {
		return nil, err
	}

	var apps []Application
	for _, app := range appMap {
		apps = append(apps, app)
	}
	sort.Sort(applications(apps))
	return apps, nil
}
