package template

import (
	"strings"

	"github.com/flosch/pongo2"
)

// custom pongo2 filters to works with
func init() {
	pongo2.RegisterFilter("header_name", HeaderNameFilter)
	pongo2.RegisterFilter("host_name", HostNameFilter)
}

func HeaderNameFilter(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	v := strings.TrimSpace(in.String())
	if v == "" {
		return pongo2.AsValue(v), nil
	}
	index := strings.Index(v, " ")
	if index < 0 {
		return pongo2.AsValue(""), nil
	}
	name := v[:index]
	return pongo2.AsValue(name), nil
}

func HostNameFilter(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	v := in.String()
	v = strings.TrimSpace(v)
	if v == "" {
		return pongo2.AsValue(v), nil
	}
	if strings.HasPrefix(v, "*") {
		v = v[1:]
		return pongo2.AsValue("hdr_end(host) -i " + v), nil
	}
	return pongo2.AsValue("hdr(host) -i " + v), nil
}
