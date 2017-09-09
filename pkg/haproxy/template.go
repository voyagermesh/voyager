package haproxy

import (
	"strings"
	"text/template"
)

func HeaderName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	index := strings.Index(v, " ")
	if index < 0 {
		return ""
	}
	return v[:index]
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

var (
	funcMap = template.FuncMap{
		"header_name": HeaderName,
		"host_name":   HostName,
	}

	haproxyTemplate *template.Template
)

func LoadTemplates(builtin, custom string) error {
	haproxyTemplate = template.New("").Funcs(funcMap)
	if builtin != "" {
		_, err := haproxyTemplate.ParseGlob(builtin)
		if err != nil {
			return err
		}
	}
	if custom != "" {
		_, err := haproxyTemplate.ParseGlob(custom)
		if err != nil {
			return err
		}
	}
	return nil
}
