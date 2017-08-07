package haproxy

import (
	"bytes"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/util/sets"
)

func RenderConfig(data TemplateData) (string, error) {
	if err := data.isValid(); err != nil {
		return "", err
	}
	data.canonicalize()
	var buf bytes.Buffer
	err := haproxyTemplate.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (td *TemplateData) canonicalize() {
	if td.DefaultBackend != nil {
		td.DefaultBackend.canonicalize()
	}
	for i := range td.HTTPService {
		svc := td.HTTPService[i]
		for j := range svc.Paths {
			svc.Paths[j].Backend.canonicalize()
		}
		sort.Slice(svc.Paths, func(i, j int) bool { return svc.Paths[i].sortKey() < svc.Paths[j].sortKey() })
	}
	sort.Slice(td.HTTPService, func(i, j int) bool { return td.HTTPService[i].sortKey() < td.HTTPService[j].sortKey() })
	sort.Slice(td.TCPService, func(i, j int) bool { return td.TCPService[i].sortKey() < td.TCPService[j].sortKey() })
	sort.Slice(td.DNSResolvers, func(i, j int) bool { return td.DNSResolvers[i].Name < td.DNSResolvers[j].Name })
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
