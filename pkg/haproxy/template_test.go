package haproxy

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/appscode/go/runtime"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
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
		MaxConnections: 3000,
		SSLRedirect:    true,
		Limit:          &Limit{Rate: 5, TimeSecond: 20},
	}
	testParsedConfig := TemplateData{
		SharedInfo: si,
		ErrorFiles: []*ErrorFile{
			{
				StatusCode: "403",
				Command:    "errorfile",
				Value:      "/srv/voyager/errorfiles/403.http",
			},
			{
				StatusCode: "402",
				Command:    "errorloc",
				Value:      "goolge.com",
			},
		},
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
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
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
						Host: "http.appscode.test",
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
				SharedInfo:    si,
				FrontendName:  "two",
				Port:          933,
				FrontendRules: []string{},
				OffloadSSL:    true,
				Paths: []*HTTPPath{
					{
						Path: "/kool",
						Backend: Backend{
							Name:         "kool",
							Sticky:       true,
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
				SharedInfo:    si,
				FrontendName:  "three",
				Port:          9334,
				FrontendRules: []string{},
				OffloadSSL:    true,
				Paths: []*HTTPPath{
					{
						Path: "/kool",
						Backend: Backend{
							Name:         "kool",
							Sticky:       true,
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
				SharedInfo:    si,
				FrontendName:  "four",
				Port:          8334,
				NodePort:      32000,
				FrontendRules: []string{},
				OffloadSSL:    true,
				Paths: []*HTTPPath{
					{
						Host: "ex.appscode.test",
						Path: "/yara",
						Backend: Backend{
							Name:   "yara",
							Sticky: true,
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "five",
				Port:          80,
				FrontendRules: []string{},
				OffloadSSL:    true,
				Paths: []*HTTPPath{
					{
						Host: "ex.appscode.test",
						Path: "/yara",
						Backend: Backend{
							Name:   "yara",
							Sticky: true,
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "http-with-frontend-rule",
				Port:          80,
				FrontendRules: []string{"rule one", "rule two"},
				Paths: []*HTTPPath{
					{
						Host: "ex.appscode.test",
						Path: "/yara",
						Backend: Backend{
							Name: "yara",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{EnableHSTS: false},
				FrontendName: "with-hsts-disabled",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &SharedInfo{EnableHSTS: true, HSTSMaxAge: 100},
				FrontendName: "with-max-age",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSIncludeSubDomains: true},
				FrontendName: "with-subdomains",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSPreload: true},
				FrontendName: "with-preload",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSIncludeSubDomains: true, HSTSPreload: true},
				FrontendName: "with-subdomains-preload",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &SharedInfo{WhitelistSourceRange: "192.168.100.1 192.168.99.100"},
				FrontendName: "with-whitelist-http",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   si,
				FrontendName: "http-with-backend-maxconn",
				Port:         80,
				Paths: []*HTTPPath{
					{
						Backend: Backend{
							Name: "backend-maxconn",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", MaxConnections: 20, Weight: 2},
								{Name: "second", IP: "10.244.2.2", Port: "2323", Weight: 5},
							},
						},
					},
				},
			},
		},
		TCPService: []*TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
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
				SharedInfo:    si,
				FrontendName:  "daemon",
				Host:          "hello.ok.domain",
				Port:          "4444",
				FrontendRules: []string{},
				CertFile:      "this-is-secret",
				PEMName:       "secret-pem",
				Backend: Backend{
					Name: "daemon",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "katherin",
				ALPNOptions:   "alpn h2options",
				Host:          "hello.ok.domain",
				Port:          "4444",
				FrontendRules: []string{},
				Backend: Backend{
					Name: "katherin",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "kate-becket",
				ALPNOptions:   "alpn h2options",
				Host:          "hello.ok.domain",
				Port:          "4444",
				FrontendRules: []string{},
				Backend: Backend{
					Name: "kate-becket",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
						{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name"},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "rick-castle",
				ALPNOptions:   "alpn h2options",
				Host:          "hello.ok.domain",
				Port:          "4445",
				FrontendRules: []string{},
				Backend: Backend{
					Name: "kate-becket",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify none"},
						{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name", TLSOption: "ssl verify required"},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "with-sticky-options",
				Host:         "hello.ok.domain",
				Port:         "4449",
				Backend: Backend{
					Name:   "kate-becket",
					Sticky: true,
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify none"},
						{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name", TLSOption: "ssl verify required"},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "with-frontend-rules",
				Port:          "4445",
				FrontendRules: []string{"rule one", "rule two"},
				Backend: Backend{
					Name: "kate-becket",
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
			{
				SharedInfo:   &SharedInfo{WhitelistSourceRange: "192.168.100.1 192.168.99.100"},
				FrontendName: "with-whitelist-tcp",
				Port:         "4446",
			},
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTemplateAuth(t *testing.T) {
	si := &SharedInfo{
		DefaultBackend: &Backend{
			Name: "default",
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
		UserLists: []UserList{
			{
				Name: "auth",
				Users: []AuthUser{
					{
						Username:  "foo",
						Password:  "#bar",
						Encrypted: true,
					},
					{
						Username:  "foo2",
						Password:  "bar2",
						Encrypted: false,
					},
				},
			},
			{
				Name: "auth2",
				Users: []AuthUser{
					{
						Username:  "foo",
						Password:  "#bar",
						Encrypted: true,
					},
					{
						Username:  "foo2",
						Password:  "bar2",
						Encrypted: false,
					},
				},
			},
		},
		HTTPService: []*HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name: "elijah",
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
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "two",
				Port:          933,
				FrontendRules: []string{},
				Paths: []*HTTPPath{
					{
						Path: "/kool",
						Backend: Backend{
							Name: "kool",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
		},
		TCPService: []*TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
				Backend: Backend{
					Name:         "stefan",
					BackendRules: []string{"first rule", "second rule"},
					Endpoints: []*Endpoint{
						{Name: "first", IP: "10.244.2.1", Port: "2323"},
						{Name: "first", IP: "10.244.2.2", Port: "2324"},
					},
				},
			},
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTemplateServiceAuth(t *testing.T) {
	si := &SharedInfo{
		DefaultBackend: &Backend{
			Name: "default",
			Endpoints: []*Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
				{Name: "first", IP: "10.244.2.2", Port: "2324"},
			},
			BasicAuth: &BasicAuth{
				Realm:     "Required",
				UserLists: []string{"auth"},
			},
		},
	}
	testParsedConfig := TemplateData{
		SharedInfo: si,
		HTTPService: []*HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name: "elijah",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
							BasicAuth: &BasicAuth{
								Realm:     "Required",
								UserLists: []string{"auth2"},
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
				},
			},
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestDefaultFrontend(t *testing.T) {
	testParsedConfig := TemplateData{
		SharedInfo: &SharedInfo{
			DefaultBackend: &Backend{
				Name: "default",
				Endpoints: []*Endpoint{
					{Name: "first", IP: "10.244.2.1", Port: "2323"},
					{Name: "second", IP: "10.244.2.2", Port: "2324"},
				},
			},
			SSLRedirect: true,
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTLSAuth(t *testing.T) {
	si := &SharedInfo{}
	testParsedConfig := TemplateData{
		SharedInfo: si,
		HTTPService: []*HTTPService{
			{
				SharedInfo:    si,
				OffloadSSL:    true,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				TLSAuth: &TLSAuth{
					VerifyClient: "required",
					Headers: map[string]string{
						"X-TEST":      "add",
						"X-TEST-NONE": "none",
					},
					ErrorPage: "google.com",
				},
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name: "elijah",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "two",
				OffloadSSL:    true,
				Port:          90,
				FrontendRules: []string{},
				TLSAuth: &TLSAuth{
					ErrorPage: "google.com",
				},
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name: "elijah",
							Endpoints: []*Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestHealthCheck(t *testing.T) {
	si := &SharedInfo{
		DefaultBackend: &Backend{
			Name: "default",
			Endpoints: []*Endpoint{
				{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
				{Name: "bbb", IP: "10.244.2.1", Port: "2323", CheckHealth: true},
				{Name: "ccc", IP: "10.244.2.1", Port: "2323", CheckHealth: true, CheckHealthPort: "5050"},
				{Name: "ddd", IP: "10.244.2.1", Port: "2323", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, CheckHealthPort: "5050"},
			},
		},
	}
	testParsedConfig := TemplateData{
		SharedInfo: si,
		HTTPService: []*HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Paths: []*HTTPPath{
					{
						Path: "/elijah",
						Backend: Backend{
							Name: "elijah",
							Endpoints: []*Endpoint{
								{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
								{Name: "bbb", IP: "10.244.2.1", Port: "2323", CheckHealth: true},
								{Name: "ccc", IP: "10.244.2.1", Port: "2323", CheckHealth: true, CheckHealthPort: "5050"},
								{Name: "ddd", IP: "10.244.2.1", Port: "2323", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, CheckHealthPort: "5050"},
							},
						},
					},
				},
			},
		},
		TCPService: []*TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
				Backend: Backend{
					Name: "stefan",
					Endpoints: []*Endpoint{
						{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
						{Name: "bbb", IP: "10.244.2.1", Port: "2323", CheckHealth: true},
						{Name: "ccc", IP: "10.244.2.1", Port: "2323", CheckHealth: true, CheckHealthPort: "5050"},
						{Name: "ddd", IP: "10.244.2.1", Port: "2323", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, CheckHealthPort: "5050"},
					},
				},
			},
		},
	}
	err := LoadTemplates(runtime.GOPath()+"/src/github.com/appscode/voyager/hack/docker/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}
