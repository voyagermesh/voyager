package lib

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// Network on Vultr account
type Network struct {
	ID           string `json:"NETWORKID"`
	RegionID     int    `json:"DCID,string"`
	Description  string `json:"description"`
	V4Subnet     string `json:"v4_subnet"`
	V4SubnetMask int    `json:"v4_subnet_mask"`
	Created      string `json:"date_created"`
}

type networks []Network

func (n networks) Len() int      { return len(n) }
func (n networks) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n networks) Less(i, j int) bool {
	// sort order: description, created
	if strings.ToLower(n[i].Description) < strings.ToLower(n[j].Description) {
		return true
	} else if strings.ToLower(n[i].Description) > strings.ToLower(n[j].Description) {
		return false
	}
	return n[i].Created < n[j].Created
}

// GetNetworks returns a list of Networks from Vultr account
func (c *Client) GetNetworks() (nets []Network, err error) {
	var netMap map[string]Network
	if err := c.get(`network/list`, &netMap); err != nil {
		return nil, err
	}

	for _, net := range netMap {
		nets = append(nets, net)
	}
	sort.Sort(networks(nets))
	return nets, nil
}

// CreateNetwork creates new Network on Vultr
func (c *Client) CreateNetwork(regionID int, description string, subnet *net.IPNet) (Network, error) {
	var net string
	var mask int
	values := url.Values{
		"DCID":        {fmt.Sprintf("%v", regionID)},
		"description": {description},
	}
	if subnet != nil && subnet.IP.To4() != nil {
		net = subnet.IP.To4().String()
		mask, _ = subnet.Mask.Size()
		values.Add("v4_subnet", net)
		values.Add("v4_subnet_mask", fmt.Sprintf("%v", mask))
	}
	var network Network
	if err := c.post(`network/create`, values, &network); err != nil {
		return Network{}, err
	}
	network.RegionID = regionID
	network.Description = description
	network.V4Subnet = net
	network.V4SubnetMask = mask

	return network, nil
}

// DeleteNetwork deletes an existing Network from Vultr account
func (c *Client) DeleteNetwork(id string) error {
	values := url.Values{
		"NETWORKID": {id},
	}

	return c.post(`network/destroy`, values, nil)
}
