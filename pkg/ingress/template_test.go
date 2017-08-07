package ingress

import (
	"bytes"
	"testing"
	"text/template"
	"github.com/stretchr/testify/assert"
)

func TestHeaderNameFilter(t *testing.T) {
	tpl := template.Must(template.New("").Funcs(funcMap).Parse(`
{{ .val | header_name }}
{{ .val2 | header_name }}
`))
	var buf bytes.Buffer
	tpl.Execute(&buf, map[string]string{
		"val":  "hello world",
		"val2": "hello   world",
	})
	exp := `
hello
hello
`
	assert.Equal(t, exp, buf.String())
}

func TestHostNameFilter(t *testing.T) {
	tpl := template.Must(template.New("").Funcs(funcMap).Parse(`
{{ .val | host_name }}
{{ .val2 | host_name }}
`))
	var buf bytes.Buffer
	tpl.Execute(&buf, map[string]string{
		"val":  "appscode.com",
		"val2": "*.appscode.com",
	})
	exp := `
hdr(host) -i appscode.com
hdr_end(host) -i .appscode.com
`
	assert.Equal(t, exp, buf.String())
}
