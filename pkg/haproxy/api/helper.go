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
	"sort"
	"strconv"
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	URLPrefix         = "/.well-known/acme-challenge/"
	ACMEResponderPort = 56791
)

func (td TemplateData) String() string {
	data, _ := json.MarshalIndent(td, "", " ")
	return string(data)
}

func (td *TemplateData) Canonicalize() {
	td.convertWildcardHostToEmpty()
	td.sort()
	td.moveAcmePathToTop()
}

func (td *TemplateData) convertWildcardHostToEmpty() {
	for i, svc := range td.HTTPService {
		for j, host := range svc.Hosts {
			if host.Host == api.MatchAll {
				host.Host = ""
			}
			svc.Hosts[j] = host // remove the acme path
		}
		td.HTTPService[i] = svc
	}
}

func (td *TemplateData) sort() {
	backends := td.countBackendNames()
	if td.DefaultBackend != nil {
		td.DefaultBackend.canonicalize(backends[td.DefaultBackend.Name] > 1, "", "", "", "")
	}
	for x := range td.HTTPService {
		svc := td.HTTPService[x]
		if svc.BasicAuth != nil {
			svc.BasicAuth.canonicalize()
		}

		sort.Slice(svc.Hosts, func(i, j int) bool {
			host_i := hostName(svc.Hosts[i].Host)
			host_rank_i := hostRank(svc.Hosts[i].Host)

			host_j := hostName(svc.Hosts[j].Host)
			host_rank_j := hostRank(svc.Hosts[j].Host)

			if host_rank_i == host_rank_j {
				return host_i > host_j
			}
			return host_rank_i > host_rank_j
		})

		for y := range svc.Hosts {
			host := svc.Hosts[y]
			for z := range host.Paths {
				if host.Paths[z].Backend != nil {
					host.Paths[z].Backend.canonicalize(
						backends[host.Paths[z].Backend.Name] > 1,
						svc.Address,
						host.Host,
						strconv.Itoa(svc.Port),
						host.Paths[z].Path,
					)
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

			svc.Hosts[y] = host
		}
		if svc.TLSAuth != nil {
			sort.Slice(svc.TLSAuth.Headers, func(i, j int) bool { return svc.TLSAuth.Headers[i].Header < svc.TLSAuth.Headers[j].Header })
		}
		td.HTTPService[x] = svc
	}

	for x := range td.TCPService {
		svc := td.TCPService[x]

		sort.Slice(svc.Hosts, func(i, j int) bool {
			host_i := hostName(svc.Hosts[i].Host)
			host_rank_i := hostRank(svc.Hosts[i].Host)

			host_j := hostName(svc.Hosts[j].Host)
			host_rank_j := hostRank(svc.Hosts[j].Host)

			if host_rank_i == host_rank_j {
				return host_i > host_j
			}
			return host_rank_i > host_rank_j
		})

		for _, host := range svc.Hosts {
			if host.Backend != nil {
				host.Backend.canonicalize(backends[host.Backend.Name] > 1, svc.Address, host.Host, svc.Port, "")
			}
		}

		if svc.TLSAuth != nil {
			sort.Slice(svc.TLSAuth.Headers, func(i, j int) bool { return svc.TLSAuth.Headers[i].Header < svc.TLSAuth.Headers[j].Header })
		}
		td.TCPService[x] = svc
	}

	sort.Slice(td.HTTPService, func(i, j int) bool { return td.HTTPService[i].sortKey() < td.HTTPService[j].sortKey() })
	sort.Slice(td.TCPService, func(i, j int) bool { return td.TCPService[i].sortKey() < td.TCPService[j].sortKey() })
	sort.Slice(td.DNSResolvers, func(i, j int) bool { return td.DNSResolvers[i].Name < td.DNSResolvers[j].Name })

	for i := range td.UserLists {
		td.UserLists[i].canonicalize()
	}
	sort.Slice(td.UserLists, func(i, j int) bool { return td.UserLists[i].Name < td.UserLists[j].Name })

	sort.Slice(td.TimeoutDefaults, func(i, j int) bool { return td.TimeoutDefaults[i].Phase < td.TimeoutDefaults[j].Phase })
	sort.Slice(td.OptionsDefaults, func(i, j int) bool { return td.OptionsDefaults[i].Option < td.OptionsDefaults[j].Option })
}

func (td *TemplateData) countBackendNames() map[string]int {
	backends := make(map[string]int)
	if td.DefaultBackend != nil {
		backends[td.DefaultBackend.Name]++
	}
	for _, svc := range td.HTTPService {
		for _, host := range svc.Hosts {
			for _, path := range host.Paths {
				if path.Backend != nil {
					backends[path.Backend.Name]++
				}
			}
		}
	}
	for _, svc := range td.TCPService {
		for _, host := range svc.Hosts {
			if host.Backend != nil {
				backends[host.Backend.Name]++
			}
		}
	}
	return backends
}

func (td *TemplateData) moveAcmePathToTop() {
	for i, svc := range td.HTTPService {
		if svc.Port != 80 {
			continue
		}
		for j, host := range svc.Hosts {
			if host.Host != "" {
				continue
			}

			var acmeHost *HTTPHost
			for k, path := range host.Paths {
				if path.Path != URLPrefix {
					continue
				}

				acmeHost = &HTTPHost{
					Host:  "",
					Paths: []*HTTPPath{path},
				}
				copy(host.Paths[k:], host.Paths[k+1:])
				host.Paths[len(host.Paths)-1] = nil // or the zero value of T
				host.Paths = host.Paths[:len(host.Paths)-1]
				break
			}

			if acmeHost != nil {
				if len(host.Paths) == 0 {
					copy(svc.Hosts[j:], svc.Hosts[j+1:])
					svc.Hosts[len(svc.Hosts)-1] = nil // or the zero value of T
					svc.Hosts = svc.Hosts[:len(svc.Hosts)-1]
				} else {
					svc.Hosts[j] = host // remove the acme path
				}

				// inject Host into 0 index
				svc.Hosts = append([]*HTTPHost{acmeHost}, svc.Hosts...)
				break
			}
		}
		td.HTTPService[i] = svc
	}
}

func (td *TemplateData) IsValid() error {
	frontends := sets.NewString()
	backends := sets.NewString()

	if td.DefaultBackend != nil {
		backends.Insert(td.DefaultBackend.Name)
	}

	for _, svc := range td.HTTPService {
		if frontends.Has(svc.FrontendName) {
			return errors.Errorf("haproxy frontend name %s is reused", svc.FrontendName)
		} else {
			frontends.Insert(svc.FrontendName)
		}

		for _, host := range svc.Hosts {
			for _, path := range host.Paths {
				if path.Backend != nil {
					if backends.Has(path.Backend.Name) {
						return errors.Errorf("haproxy backend name %s is reused", path.Backend.Name)
					} else {
						backends.Insert(path.Backend.Name)
					}
				}
			}
		}
	}

	for _, svc := range td.TCPService {
		if frontends.Has(svc.FrontendName) {
			return errors.Errorf("haproxy frontend name %s is reused", svc.FrontendName)
		} else {
			frontends.Insert(svc.FrontendName)
		}

		for _, host := range svc.Hosts {
			if host.Backend != nil {
				if backends.Has(host.Backend.Name) {
					return errors.Errorf("haproxy backend name %s is reused", host.Backend.Name)
				} else {
					backends.Insert(host.Backend.Name)
				}
			}
		}
	}
	return nil
}

func hostName(host string) string {
	if host == "" {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(host, "*."))
}

func hostRank(host string) int {
	if host == "" {
		return 0
	}
	if strings.HasPrefix(host, "*") {
		return 1
	}
	return 2
}

func TimeOutConfigs(in map[string]string) []TimeoutConfig {
	var out []TimeoutConfig
	for k, v := range in {
		out = append(out, TimeoutConfig{
			Phase:    k,
			Duration: v,
		})
	}
	return out
}

func OptionConfigs(in map[string]bool) []OptionConfig {
	var out []OptionConfig
	for k, v := range in {
		out = append(out, OptionConfig{
			Option:  k,
			Enabled: v,
		})
	}
	return out
}

func TLSHeaders(in map[string]string) []TLSHeader {
	var out []TLSHeader
	for k, v := range in {
		out = append(out, TLSHeader{
			Header: k,
			Value:  v,
		})
	}
	return out
}
