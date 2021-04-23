/*
Copyright AppsCode Inc. and Contributors

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
				host.Paths[z].Backend.canonicalize(false, "", host.Host, "80", host.Paths[z].Path)
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

func TestTLSHeaders(t *testing.T) {
	td := TemplateData{
		SharedInfo: &SharedInfo{},
		HTTPService: []*HTTPService{
			{
				TLSAuth: &TLSAuth{
					Headers: []TLSHeader{
						{"h2", ""},
						{"h3", ""},
						{"h1", ""},
					},
				},
			},
			{
				TLSAuth: &TLSAuth{
					Headers: []TLSHeader{
						{"h5", ""},
						{"h6", ""},
						{"h4", ""},
					},
				},
			},
		},
		TCPService: []*TCPService{
			{
				TLSAuth: &TLSAuth{
					Headers: []TLSHeader{
						{"h2", ""},
						{"h3", ""},
						{"h1", ""},
					},
				},
			},
			{
				TLSAuth: &TLSAuth{
					Headers: []TLSHeader{
						{"h5", ""},
						{"h6", ""},
						{"h4", ""},
					},
				},
			},
		},
	}

	td.sort()
	for _, svc := range td.HTTPService {
		fmt.Println(svc.TLSAuth.Headers)
	}
	for _, svc := range td.TCPService {
		fmt.Println(svc.TLSAuth.Headers)
	}
}

func TestTimeOutConfigs(t *testing.T) {
	td := TemplateData{
		SharedInfo: &SharedInfo{},
		TimeoutDefaults: []TimeoutConfig{
			{"p2", ""},
			{"p3", ""},
			{"p1", ""},
		},
	}
	td.sort()
	fmt.Println(td.TimeoutDefaults)
}

func TestConnectionModes(t *testing.T) {
	td := TemplateData{
		SharedInfo: &SharedInfo{},
		OptionsDefaults: []OptionConfig{
			{"p2", false},
			{"p3", false},
			{"p1", false},
		},
	}
	td.sort()
	fmt.Println(td.OptionsDefaults)
}
