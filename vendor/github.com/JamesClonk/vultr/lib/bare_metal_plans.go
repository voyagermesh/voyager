package lib

import (
	"fmt"
	"sort"
)

// BareMetalPlan is a bare metal-compatible plan on Vultr.
type BareMetalPlan struct {
	Deprecated bool   `json:"deprecated"`
	ID         int    `json:"METALPLANID,string"`
	Name       string `json:"name"`
	CPUs       int    `json:"cpu_count"`
	RAM        int    `json:"ram"`
	Disk       string `json:"disk"`
	Bandwidth  int    `json:"bandwidth_tb"`
	Price      int    `json:"price_per_month"`
	Regions    []int  `json:"available_locations"`
	Type       string `json:"type"`
}

type bareMetalPlans []BareMetalPlan

func (b bareMetalPlans) Len() int      { return len(b) }
func (b bareMetalPlans) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b bareMetalPlans) Less(i, j int) bool {
	ba := b[i].Price
	bb := b[j].Price
	ra := b[i].RAM
	rb := b[j].RAM
	da := b[i].Disk
	db := b[j].Disk

	// sort order: price, vcpu, ram, disk
	if ba < bb {
		return true
	} else if ba > bb {
		return false
	}

	if b[i].CPUs < b[j].CPUs {
		return true
	} else if b[i].CPUs > b[j].CPUs {
		return false
	}

	if ra < rb {
		return true
	} else if ra > rb {
		return false
	}

	return da < db
}

// GetBareMetalPlans returns a list of all available bare metal plans on Vultr account.
func (c *Client) GetBareMetalPlans() ([]BareMetalPlan, error) {
	var bareMetalPlanMap map[string]BareMetalPlan
	if err := c.get(`plans/list_baremetal`, &bareMetalPlanMap); err != nil {
		return nil, err
	}

	var b bareMetalPlans
	for _, bareMetalPlan := range bareMetalPlanMap {
		b = append(b, bareMetalPlan)
	}

	sort.Sort(bareMetalPlans(b))
	return b, nil
}

// GetAvailableBareMetalPlansForRegion returns available bare metal plans for specified region.
func (c *Client) GetAvailableBareMetalPlansForRegion(id int) ([]int, error) {
	var bareMetalPlanIDs []int
	if err := c.get(fmt.Sprintf(`regions/availability?DCID=%v`, id), &bareMetalPlanIDs); err != nil {
		return nil, err
	}
	return bareMetalPlanIDs, nil
}
