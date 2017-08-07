package ingress

import (
	"strings"

	"github.com/flosch/pongo2"
)

// custom pongo2 filters to works with
func init() {
	pongo2.RegisterFilter("header_name", HeaderName)
	pongo2.RegisterFilter("host_name", HostName)
}

func HeaderName(v string) string {
	v := strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	index := strings.Index(v, " ")
	if index < 0 {
		return pongo2.AsValue(""), nil
	}
	name := v[:index]
	return pongo2.AsValue(name), nil
}

func HostName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "*") {
		return "hdr_end(host) -i " + v[1:]
	}
	return "hdr(host) -i " + v
}
