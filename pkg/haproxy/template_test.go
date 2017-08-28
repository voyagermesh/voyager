package haproxy

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/appscode/voyager/api"
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

func TestTemplate(t *testing.T) {
	si := &SharedInfo{
		DefaultBackend: &Backend{
			Name:         "default",
			BackendRules: []string{"first rule", "second rule"},
			RewriteRules: []string{"first rule", "second rule"},
			HeaderRules:  []string{"firstName value", "secondName value"},
			Endpoints: []*Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
				{Name: "first", IP: "10.244.2.2", Port: "2324"},
			},
		},
	}
	testParsedConfig := TemplateData{
		SharedInfo: si,
		TimeoutDefaults: map[string]string{
			"client": "2s",
			"fin":    "1d",
		},
		OptionsDefaults: map[string]bool{
			"full-value":     true,
			"full-value-two": true,
			"with-no":        false,
			"with-no-two":    false,
		},
		Stats: &StatsInfo{Port: 1234},
		DNSResolvers: []*api.DNSResolver{
			{Name: "first", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, Timeout: map[string]string{"client": "5s", "fin": "1d"}, Hold: map[string]string{"client": "5s", "fin": "1d"}},
			{Name: "second", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, CheckHealth: true, Hold: map[string]string{"client": "5s", "fin": "1d"}},
			{Name: "third", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, CheckHealth: true},
		},
		HTTPService: []*HTTPService{
			{
				SharedInfo:   si,
				FrontendName: "one",
				Port:         80,
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name:         "elijah",
							BackendRules: []string{"first rule", "second rule"},
							RewriteRules: []string{"first rule", "second rule"},
							HeaderRules:  []string{"firstName value", "secondName value"},
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
					{
						Path: "/nicklause",
						Backend: Backend{
							Name: "nicklause",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324", CheckHealth: true},
							},
						},
					},
					{
						Host: "test.appscode.dev",
						Path: "/rebeka",
						Backend: Backend{
							Name:         "rebecka",
							RewriteRules: []string{"first rule", "second rule"},
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{Sticky: true},
				FrontendName: "two",
				Port:         933,
				UsesSSL:      true,
				Paths: []*HTTPPath{
					{
						Path: "/kool",
						Backend: Backend{
							Name:         "kool",
							BackendRules: []string{"first rule", "second rule"},
							RewriteRules: []string{"first rule", "second rule"},
							HeaderRules:  []string{"firstName value", "secondName value"},
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{Sticky: true},
				FrontendName: "three",
				Port:         9334,
				UsesSSL:      true,
				Paths: []*HTTPPath{
					{
						Path: "/kool",
						Backend: Backend{
							Name:         "kool",
							BackendRules: []string{"first rule", "second rule"},
							RewriteRules: []string{"first rule", "second rule"},
							HeaderRules:  []string{"firstName value", "secondName value"},
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
								{Name: "first", IP: "10.244.2.2", Port: "2324", TLSOption: "ssl verify none"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{Sticky: true},
				FrontendName: "four",
				Port:         8334,
				NodePort:     32000,
				UsesSSL:      true,
				Paths: []*HTTPPath{
					{
						Host: "ex.appscode.dev",
						Path: "/yara",
						Backend: Backend{
							Name: "yara",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{Sticky: true},
				FrontendName: "five",
				Port:         80,
				UsesSSL:      true,
				Paths: []*HTTPPath{
					{
						Host: "ex.appscode.dev",
						Path: "/yara",
						Backend: Backend{
							Name: "yara",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
		},
		TCPService: []*TCPService{
			{
				SharedInfo:   si,
				FrontendName: "stefan",
				Port:         "333",
				Backend: Backend{
					Name:         "stefan",
					BackendRules: []string{"first rule", "second rule"},
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "daemon",
				Host:         "hello.ok.domain",
				Port:         "4444",
				SecretName:   "this-is-secret",
				PEMName:      "secret-pem",
				Backend: Backend{
					Name: "daemon",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "katherin",
				ALPNOptions:  "alpn h2options",
				Host:         "hello.ok.domain",
				Port:         "4444",
				Backend: Backend{
					Name: "katherin",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "kate-becket",
				ALPNOptions:  "alpn h2options",
				Host:         "hello.ok.domain",
				Port:         "4444",
				Backend: Backend{
					Name: "kate-becket",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
						{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name"},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "rick-castle",
				ALPNOptions:  "alpn h2options",
				Host:         "hello.ok.domain",
				Port:         "4445",
				Backend: Backend{
					Name: "kate-becket",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify none"},
						{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name", TLSOption: "ssl verify required"},
					},
				},
			},
		},
	}
	config, err := RenderConfig(testParsedConfig)
	assert.Nil(t, err)
	if testing.Verbose() {
		fmt.Println(err, "\n", config)
	}
}
