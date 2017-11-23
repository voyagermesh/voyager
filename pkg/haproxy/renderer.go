package haproxy

import (
	"bytes"
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

func (td *TemplateData) canonicalize() {
	if td.DefaultBackend != nil {
		td.DefaultBackend.canonicalize()
	}
	for i := range td.HTTPService {
		svc := td.HTTPService[i]
		if svc.BasicAuth != nil {
			svc.BasicAuth.canonicalize()
		}
		sort.Slice(svc.Paths, func(j, k int) bool {
			host_j := hostName(svc.Paths[j].Host)
			host_rank_j := hostRank(svc.Paths[j].Host)
			path_j := strings.ToLower(strings.Trim(svc.Paths[j].Path, "/"))
			path_comp_j := len(strings.Split(path_j, "/"))

			host_k := hostName(svc.Paths[k].Host)
			host_rank_k := hostRank(svc.Paths[k].Host)
			path_k := strings.ToLower(strings.Trim(svc.Paths[k].Path, "/"))
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
		for j := range svc.Paths {
			svc.Paths[j].Backend.canonicalize()
		}
		td.HTTPService[i] = svc
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

		for _, path := range svc.Paths {
			if backends.Has(path.Backend.Name) {
				return fmt.Errorf("HAProxy backend name %s is reused.", path.Backend.Name)
			} else {
				frontends.Insert(svc.FrontendName)
			}
		}
	}

	for _, svc := range td.TCPService {
		if frontends.Has(svc.FrontendName) {
			return fmt.Errorf("HAProxy frontend name %s is reused.", svc.FrontendName)
		} else {
			frontends.Insert(svc.FrontendName)
		}

		if backends.Has(svc.Backend.Name) {
			return fmt.Errorf("HAProxy backend name %s is reused.", svc.Backend.Name)
		} else {
			frontends.Insert(svc.FrontendName)
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
