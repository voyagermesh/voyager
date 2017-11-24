package haproxy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestPathOrdering(t *testing.T) {
	paths := []HTTPPath{
		{Host: "abc.com", Path: "/"},
		{Host: "abc.com", Path: "/T"},
		{Host: "", Path: "/t"},
		{Host: "*.*.abc.com", Path: "/Ta"},
		{Host: "", Path: "/TA"},
	}

	sort.Slice(paths, func(j, k int) bool {
		host_j := hostName(paths[j].Host)
		host_rank_j := hostRank(paths[j].Host)
		path_j := strings.ToLower(strings.Trim(paths[j].Path, "/"))
		path_comp_j := len(strings.Split(path_j, "/"))

		host_k := hostName(paths[k].Host)
		host_rank_k := hostRank(paths[k].Host)
		path_k := strings.ToLower(strings.Trim(paths[k].Path, "/"))
		path_comp_k := len(strings.Split(path_k, "/"))

		if host_rank_j == host_rank_k {
			if host_j == host_k {
				if path_comp_j == path_comp_k {
					return path_j > path_k
				}
				return path_comp_j > path_comp_k
			}
			return host_j > host_k
		}
		return host_rank_j > host_rank_k
	})

	b, _ := json.MarshalIndent(paths, "", "  ")
	fmt.Println(string(b))
}
