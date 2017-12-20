package haproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/appscode/go/log"
	"k8s.io/apimachinery/pkg/util/sets"
)

func RenderConfig(data TemplateData) (string, error) {
	if err := data.isValid(); err != nil {
		return "", err
	}
	data.canonicalize()

	var buf bytes.Buffer
	err := haproxyTemplate.ExecuteTemplate(&buf, "haproxy.cfg", data)
	if err != nil {
		log.Error(err)
		return "", err
	}
	lines := strings.Split(buf.String(), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n"), nil
}

func (td TemplateData) String() string {
	data, _ := json.MarshalIndent(td, "", " ")
	return string(data)
}

func (td *TemplateData) canonicalize() {
	if td.DefaultBackend != nil {
		td.DefaultBackend.canonicalize()
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
					host.Paths[z].Backend.canonicalize()
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

		td.HTTPService[x] = svc
	}
	sort.Slice(td.HTTPService, func(i, j int) bool { return td.HTTPService[i].sortKey() < td.HTTPService[j].sortKey() })
	sort.Slice(td.TCPService, func(i, j int) bool { return td.TCPService[i].sortKey() < td.TCPService[j].sortKey() })
	sort.Slice(td.DNSResolvers, func(i, j int) bool { return td.DNSResolvers[i].Name < td.DNSResolvers[j].Name })

	for i := range td.UserLists {
		td.UserLists[i].canonicalize()
	}
	sort.Slice(td.UserLists, func(i, j int) bool { return td.UserLists[i].Name < td.UserLists[j].Name })
}

func (td *TemplateData) isValid() error {
	frontends := sets.NewString()
	backends := sets.NewString()

	if td.DefaultBackend != nil {
		backends.Insert(td.DefaultBackend.Name)
	}

	for _, svc := range td.HTTPService {
		if frontends.Has(svc.FrontendName) {
			return fmt.Errorf("HAProxy frontend name %s is reused.", svc.FrontendName)
		} else {
			frontends.Insert(svc.FrontendName)
		}

		for _, host := range svc.Hosts {
			for _, path := range host.Paths {
				if path.Backend != nil {
					if backends.Has(path.Backend.Name) {
						return fmt.Errorf("HAProxy backend name %s is reused.", path.Backend.Name)
					} else {
						backends.Insert(path.Backend.Name)
					}
				}
			}
		}
	}

	for _, svc := range td.TCPService {
		if frontends.Has(svc.FrontendName) {
			return fmt.Errorf("HAProxy frontend name %s is reused.", svc.FrontendName)
		} else {
			frontends.Insert(svc.FrontendName)
		}

		if svc.Backend != nil {
			if backends.Has(svc.Backend.Name) {
				return fmt.Errorf("HAProxy backend name %s is reused.", svc.Backend.Name)
			} else {
				backends.Insert(svc.Backend.Name)
			}
		}
	}
	return nil
}

func hostName(host string) string {
	if host == "" || host == `*` {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(host, "*."))
}

func hostRank(host string) int {
	if host == "" || host == `*` {
		return 0
	}
	if strings.HasPrefix(host, "*") {
		return 1
	}
	return 2
}
