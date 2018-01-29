package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestPathOrdering(t *testing.T) {
	hosts := []*HTTPHost{
		{
			Host: "*.*.abc.com",
			Paths: []*HTTPPath{
				{
					Path: "/Ta",
				},
			},
		},
		{
			Host: "abc.com",
			Paths: []*HTTPPath{
				{
					Path: "/",
				},
				{
					Path: "/T",
				},
			},
		},
		{
			Host: "",
			Paths: []*HTTPPath{
				{
					Path: "/t",
				},
				{
					Path: "/TA",
				},
			},
		},
	}

	sort.Slice(hosts, func(i, j int) bool {
		host_i := hostName(hosts[i].Host)
		host_rank_i := hostRank(hosts[i].Host)

		host_j := hostName(hosts[j].Host)
		host_rank_j := hostRank(hosts[j].Host)

		if host_rank_i == host_rank_j {
			return host_i > host_j
		}
		return host_rank_i > host_rank_j
	})

	for y := range hosts {
		host := hosts[y]
		for z := range host.Paths {
			if host.Paths[z].Backend != nil {
				host.Paths[z].Backend.canonicalize(false, host.Host, "80", host.Paths[z].Path)
			}
		}

		sort.Slice(host.Paths, func(i, j int) bool {
			path_i := strings.ToLower(strings.Trim(host.Paths[i].Path, "/"))
			path_comp_i := len(strings.Split(path_i, "/"))

			path_j := strings.ToLower(strings.Trim(host.Paths[j].Path, "/"))
			path_comp_j := len(strings.Split(path_j, "/"))

			if path_comp_i == path_comp_j {
				return path_i > path_j
			}
			return path_comp_i > path_comp_j
		})

		hosts[y] = host
	}

	b, _ := json.MarshalIndent(hosts, "", "  ")
	fmt.Println(string(b))
}
