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

package template

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	hpi "voyagermesh.dev/voyager/pkg/haproxy/api"

	"github.com/stretchr/testify/assert"
)

func TestHeaderNameFilter(t *testing.T) {
	tpl := template.Must(template.New("").Funcs(funcMap).Parse(`
{{ .val | header_name }}
{{ .val2 | header_name }}
`))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, map[string]string{
		"val":  "hello world",
		"val2": "hello   world",
	})
	if err != nil {
		t.Fatal(err)
	}
	exp := `
hello
hello
`
	assert.Equal(t, exp, buf.String())
}

func TestTemplate(t *testing.T) {
	si := &hpi.SharedInfo{
		DefaultBackend: &hpi.Backend{
			Name:         "default",
			BackendRules: []string{"first rule", "second rule"},
			RewriteRules: []string{"first rule", "second rule"},
			HeaderRules:  []string{"firstName value", "secondName value"},
			Endpoints: []*hpi.Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
				{Name: "first", IP: "10.244.2.2", Port: "2324"},
			},
		},
		MaxConnections: 3000,
		Limit:          &hpi.Limit{Rate: 5, TimeSecond: 20},
	}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		ErrorFiles: []*hpi.ErrorFile{
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
		TimeoutDefaults: []hpi.TimeoutConfig{
			{Phase: "client", Duration: "2s"},
			{Phase: "fin", Duration: "1d"},
		},
		OptionsDefaults: []hpi.OptionConfig{
			{Option: "full-value", Enabled: true},
			{Option: "full-value-two", Enabled: true},
			{Option: "with-no"},
			{Option: "with-no-two"},
		},
		Stats: &hpi.StatsInfo{Port: 1234},
		DNSResolvers: []*api.DNSResolver{
			{Name: "first", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, Timeout: map[string]string{"client": "5s", "fin": "1d"}, Hold: map[string]string{"client": "5s", "fin": "1d"}},
			{Name: "second", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, CheckHealth: true, Hold: map[string]string{"client": "5s", "fin": "1d"}},
			{Name: "third", NameServer: []string{"foo:54", "bar:53"}, Retries: 5, CheckHealth: true},
		},
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name:         "elijah",
									BackendRules: []string{"first rule", "second rule"},
									RewriteRules: []string{"first rule", "second rule"},
									HeaderRules:  []string{"firstName value", "secondName value"},
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
							},
							{
								Path: "/nicklause",
								Backend: &hpi.Backend{
									Name: "nicklause",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324", CheckHealth: true},
									},
								},
							},
						},
					},
					{
						Host: "http.appscode.test",
						Paths: []*hpi.HTTPPath{
							{
								Path: "/rebeka",
								Backend: &hpi.Backend{
									Name:         "rebecka",
									RewriteRules: []string{"first rule", "second rule"},
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, TLSOption: "ssl verify required"},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/kool",
								Backend: &hpi.Backend{
									Name:         "kool1",
									Sticky:       true,
									BackendRules: []string{"first rule", "second rule"},
									RewriteRules: []string{"first rule", "second rule"},
									HeaderRules:  []string{"firstName value", "secondName value"},
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/kool",
								Backend: &hpi.Backend{
									Name:         "kool2",
									Sticky:       true,
									BackendRules: []string{"first rule", "second rule"},
									RewriteRules: []string{"first rule", "second rule"},
									HeaderRules:  []string{"firstName value", "secondName value"},
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
										{Name: "first", IP: "10.244.2.2", Port: "2324", TLSOption: "ssl verify none"},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Host: "ex.appscode.test",
						Paths: []*hpi.HTTPPath{
							{
								Path: "/yara",
								Backend: &hpi.Backend{
									Name:   "yara1",
									Sticky: true,
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Host: "ex.appscode.test",
						Paths: []*hpi.HTTPPath{
							{
								Path: "/yara",
								Backend: &hpi.Backend{
									Name:   "yara2",
									Sticky: true,
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify required"},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Host: "ex.appscode.test",
						Paths: []*hpi.HTTPPath{
							{
								Path: "/yara",
								Backend: &hpi.Backend{
									Name: "yara3",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &hpi.SharedInfo{EnableHSTS: false},
				FrontendName: "with-hsts-disabled",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &hpi.SharedInfo{EnableHSTS: true, HSTSMaxAge: 100},
				FrontendName: "with-max-age",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &hpi.SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSIncludeSubDomains: true},
				FrontendName: "with-subdomains",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &hpi.SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSPreload: true},
				FrontendName: "with-preload",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &hpi.SharedInfo{EnableHSTS: true, HSTSMaxAge: 100, HSTSIncludeSubDomains: true, HSTSPreload: true},
				FrontendName: "with-subdomains-preload",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   &hpi.SharedInfo{WhitelistSourceRange: "192.168.100.1 192.168.99.100"},
				FrontendName: "with-whitelist-http",
				OffloadSSL:   true,
			},
			{
				SharedInfo:   si,
				FrontendName: "http-with-backend-maxconn",
				Port:         80,
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Backend: &hpi.Backend{
									Name: "backend-maxconn",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", MaxConnections: 20, Weight: 2},
										{Name: "second", IP: "10.244.2.2", Port: "2323", Weight: 5},
									},
								},
							},
						},
					},
				},
			},
		},
		TCPService: []*hpi.TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name:         "stefan",
							BackendRules: []string{"first rule", "second rule"},
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "daemon",
				Port:          "4444",
				FrontendRules: []string{},
				OffloadSSL:    true,
				Hosts: []*hpi.TCPHost{
					{
						Host: "hello.ok.domain",
						Backend: &hpi.Backend{
							Name: "daemon",
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "katherin",
				ALPNOptions:   "alpn h2options",
				Port:          "4444",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Host: "hello.ok.domain",
						Backend: &hpi.Backend{
							Name: "katherin",
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "kate-becket",
				ALPNOptions:   "alpn h2options",
				Port:          "4444",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Host: "hello.ok.domain",
						Backend: &hpi.Backend{
							Name: "kate-becket1",
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
								{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "rick-castle",
				ALPNOptions:   "alpn h2options",
				Port:          "4445",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Host: "hello.ok.domain",
						Backend: &hpi.Backend{
							Name: "kate-becket2",
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify none"},
								{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name", TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "with-sticky-options",
				Port:         "4449",
				Hosts: []*hpi.TCPHost{
					{
						Host: "hello.ok.domain",
						Backend: &hpi.Backend{
							Name:   "kate-becket3",
							Sticky: true,
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true, TLSOption: "ssl verify none"},
								{Name: "first", IP: "10.244.2.2", Port: "2324", ExternalName: "ext-name", TLSOption: "ssl verify required"},
							},
						},
					},
				},
			},
			{
				SharedInfo:    si,
				FrontendName:  "with-frontend-rules",
				Port:          "4445",
				FrontendRules: []string{"rule one", "rule two"},
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name: "kate-becket4",
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
			{
				SharedInfo:   &hpi.SharedInfo{WhitelistSourceRange: "192.168.100.1 192.168.99.100"},
				FrontendName: "with-whitelist-tcp",
				Port:         "4446",
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTemplateAuth(t *testing.T) {
	si := &hpi.SharedInfo{
		DefaultBackend: &hpi.Backend{
			Name: "default",
			Endpoints: []*hpi.Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
				{Name: "first", IP: "10.244.2.2", Port: "2324"},
			},
		},
	}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		TimeoutDefaults: []hpi.TimeoutConfig{
			{Phase: "client", Duration: "2s"},
			{Phase: "fin", Duration: "1d"},
		},
		UserLists: []hpi.UserList{
			{
				Name: "auth",
				Users: []hpi.AuthUser{
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
				Users: []hpi.AuthUser{
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
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name: "elijah",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
							},
							{
								Path: "/nicklause",
								Backend: &hpi.Backend{
									Name: "nicklause",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324", CheckHealth: true},
									},
								},
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
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/kool",
								Backend: &hpi.Backend{
									Name: "kool",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323", UseDNSResolver: true},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
							},
						},
					},
				},
			},
		},
		TCPService: []*hpi.TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name:         "stefan",
							BackendRules: []string{"first rule", "second rule"},
							Endpoints: []*hpi.Endpoint{
								{Name: "first", IP: "10.244.2.1", Port: "2323"},
								{Name: "first", IP: "10.244.2.2", Port: "2324"},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTemplateServiceAuth(t *testing.T) {
	si := &hpi.SharedInfo{
		DefaultBackend: &hpi.Backend{
			Name: "default",
			Endpoints: []*hpi.Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
				{Name: "first", IP: "10.244.2.2", Port: "2324"},
			},
			BasicAuth: &hpi.BasicAuth{
				Realm:     "Required",
				UserLists: []string{"auth"},
			},
		},
	}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name: "elijah",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
									BasicAuth: &hpi.BasicAuth{
										Realm:     "Required",
										UserLists: []string{"auth2"},
									},
								},
							},
							{
								Path: "/nicklause",
								Backend: &hpi.Backend{
									Name: "nicklause",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324", CheckHealth: true},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestDefaultFrontend(t *testing.T) {
	testParsedConfig := hpi.TemplateData{
		SharedInfo: &hpi.SharedInfo{
			DefaultBackend: &hpi.Backend{
				Name: "default",
				Endpoints: []*hpi.Endpoint{
					{Name: "first", IP: "10.244.2.1", Port: "2323"},
					{Name: "second", IP: "10.244.2.2", Port: "2324"},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTLSAuth(t *testing.T) {
	si := &hpi.SharedInfo{}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				OffloadSSL:    true,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				TLSAuth: &hpi.TLSAuth{
					VerifyClient: "required",
					Headers: []hpi.TLSHeader{
						{Header: "X-TEST", Value: "add"},
						{Header: "X-TEST-NONE", Value: "none"},
					},
					ErrorPage: "google.com",
				},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name: "elijah1",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
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
				TLSAuth: &hpi.TLSAuth{
					ErrorPage: "google.com",
				},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name: "elijah2",
									Endpoints: []*hpi.Endpoint{
										{Name: "first", IP: "10.244.2.1", Port: "2323"},
										{Name: "first", IP: "10.244.2.2", Port: "2324"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestHealthCheck(t *testing.T) {
	si := &hpi.SharedInfo{
		DefaultBackend: &hpi.Backend{
			Name: "default",
			Endpoints: []*hpi.Endpoint{
				{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
				{Name: "bbb", IP: "10.244.2.1", Port: "2323", CheckHealth: true},
				{Name: "ccc", IP: "10.244.2.1", Port: "2323", CheckHealth: true, CheckHealthPort: "5050"},
				{Name: "ddd", IP: "10.244.2.1", Port: "2323", ExternalName: "name", DNSResolver: "one", UseDNSResolver: true, CheckHealth: true, CheckHealthPort: "5050"},
			},
		},
	}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Path: "/elijah",
								Backend: &hpi.Backend{
									Name: "elijah",
									Endpoints: []*hpi.Endpoint{
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
			},
		},
		TCPService: []*hpi.TCPService{
			{
				SharedInfo:    si,
				FrontendName:  "stefan",
				Port:          "333",
				FrontendRules: []string{},
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name: "stefan",
							Endpoints: []*hpi.Endpoint{
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
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestTcpSni(t *testing.T) {
	si := &hpi.SharedInfo{}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		TCPService: []*hpi.TCPService{
			{
				SharedInfo:   si,
				FrontendName: "no-sni",
				Port:         "8080",
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{Name: "no-sni"},
						Host:    "http.voyager.test",
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "wildcard-without-tls",
				Port:         "8080",
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{Name: "wildcard-without-tls-1"},
						Host:    "*.voyager.test",
					},
					{
						Backend: &hpi.Backend{Name: "wildcard-without-tls-2"},
						Host:    "http.voyager.test",
					},
				},
			},
			{
				SharedInfo:   si,
				FrontendName: "wildcard-with-tls",
				Port:         "8080",
				OffloadSSL:   true,
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{Name: "wildcard-with-tls-1"},
						Host:    "*.voyager.test",
					},
					{
						Backend: &hpi.Backend{Name: "wildcard-with-tls-2"},
						Host:    "http.voyager.test",
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestOauth(t *testing.T) {
	si := &hpi.SharedInfo{}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				SharedInfo:    si,
				FrontendName:  "one",
				Port:          80,
				FrontendRules: []string{},
				Hosts: []*hpi.HTTPHost{
					{
						Host: "voyager.appscode.test",
						ExternalAuth: &hpi.ExternalAuth{
							AuthBackend:     "auth-be",
							AuthBackendPath: "/oauth2",
							AuthPath:        "/oauth2/auth",
							SigninPath:      "/oauth2/start",
							Paths:           []string{"/app", "/foo"},
						},
						Paths: []*hpi.HTTPPath{
							{
								Path: "/app",
								Backend: &hpi.Backend{
									Name: "app",
									Endpoints: []*hpi.Endpoint{
										{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
										{Name: "bbb", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
							{
								Path: "/app-2",
								Backend: &hpi.Backend{
									Name: "app-2",
									Endpoints: []*hpi.Endpoint{
										{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
										{Name: "bbb", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
							{
								Path: "/oauth2",
								Backend: &hpi.Backend{
									Name: "auth-be",
									Endpoints: []*hpi.Endpoint{
										{Name: "ccc", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestProtoAndALPN(t *testing.T) {
	si := &hpi.SharedInfo{}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				Port:         1111,
				SharedInfo:   si,
				FrontendName: "with-alpn-1",
				ALPNOptions:  "alpn h2,http/1.1",
				OffloadSSL:   true,
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Backend: &hpi.Backend{
									Name:  "bk-1",
									Proto: "h2",
									Endpoints: []*hpi.Endpoint{
										{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
						},
					},
				},
			},
			{
				Port:         2222,
				SharedInfo:   si,
				FrontendName: "with-alpn-2",
				Proto:        "h2",
				OffloadSSL:   true,
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Backend: &hpi.Backend{
									Name:        "bk-2",
									ALPNOptions: "alpn http/1.1,http/1.0",
									Endpoints: []*hpi.Endpoint{
										{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
									},
								},
							},
						},
					},
				},
			},
		},
		TCPService: []*hpi.TCPService{
			{
				Port:         "3333",
				SharedInfo:   si,
				FrontendName: "with-alpn-3",
				ALPNOptions:  "alpn h2,http/1.1",
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name:  "bk-3",
							Proto: "h2",
							Endpoints: []*hpi.Endpoint{
								{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
							},
						},
					},
				},
			},
			{
				Port:         "4444",
				SharedInfo:   si,
				FrontendName: "with-alpn-4",
				Proto:        "h2",
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name:        "bk-4",
							ALPNOptions: "alpn http/1.1,http/1.0",
							Endpoints: []*hpi.Endpoint{
								{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
							},
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}

func TestBackendLoadBalance(t *testing.T) {
	si := &hpi.SharedInfo{
		DefaultBackend: &hpi.Backend{
			Name: "default",
			Endpoints: []*hpi.Endpoint{
				{Name: "first", IP: "10.244.2.1", Port: "2323"},
			},
			LoadBalanceOn: "leastconn",
		},
	}
	testParsedConfig := hpi.TemplateData{
		SharedInfo: si,
		HTTPService: []*hpi.HTTPService{
			{
				Port:         1111,
				SharedInfo:   si,
				FrontendName: "http-fe",
				Hosts: []*hpi.HTTPHost{
					{
						Paths: []*hpi.HTTPPath{
							{
								Backend: &hpi.Backend{
									Name: "http-bk",
									Endpoints: []*hpi.Endpoint{
										{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
									},
									LoadBalanceOn: "leastconn",
								},
							},
						},
					},
				},
			},
		},
		TCPService: []*hpi.TCPService{
			{
				Port:         "2222",
				SharedInfo:   si,
				FrontendName: "tcp-fe",
				Hosts: []*hpi.TCPHost{
					{
						Backend: &hpi.Backend{
							Name: "tcp-bk",
							Endpoints: []*hpi.Endpoint{
								{Name: "aaa", IP: "10.244.2.1", Port: "2323"},
							},
							LoadBalanceOn: "leastconn",
						},
					},
				},
			},
		},
	}
	err := LoadTemplates("/srv/voyager/templates/*.cfg", "")
	if assert.Nil(t, err) {
		config, err := RenderConfig(testParsedConfig)
		assert.Nil(t, err)
		if testing.Verbose() {
			fmt.Println(err, "\n", config)
		}
	}
}
