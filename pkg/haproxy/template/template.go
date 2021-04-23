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
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
)

/*
<aclname>: name of the acl, usually tries to describe it as much as possible.

It must be formed from upper and lower case letters, digits, ‘-‘ (dash), ‘_’ (underscore) , ‘.’ (dot) and ‘:’ (colon).
It is case sensitive, so my_acl and My_Acl are two different ACLs.

ref: https://www.haproxy.com/documentation/aloha/7-0/haproxy/acls/
*/
func ACLName(v string) string {
	v = strings.TrimPrefix(v, "/")
	v = strings.TrimSuffix(v, "/")
	v = strings.Replace(v, "/", "-", -1)
	v = strings.Replace(v, "*", ".", -1)
	return v
}

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

func HostACLs(host string, port int, nodePort int32, useNodePort bool) []string {
	var conditions []string
	host = strings.TrimSpace(host)

	if port <= 0 {
		panic(fmt.Sprintf("port expected to be > 0, found %d", port))
	}
	if useNodePort && nodePort <= 0 {
		panic(fmt.Sprintf("nodeport expected to be > 0, found %d. must wait for nodeport assignment", nodePort))
	}
	if host == api.MatchAll {
		panic("wildcard host must be empty, found *")
	}

	if host == `` { // for wildcard domain, host is expected to be ``
		return conditions
	}

	if useNodePort {
		conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, nodePort)))
	} else {
		if port != 80 && port != 443 { // non standard http ports
			conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, port)))
		} else { // http or https
			conditions = append(conditions, hostMatcher(host))
			conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, port)))
		}
	}
	return conditions
}

func hostMatcher(v string) string {
	if strings.HasPrefix(v, "*") {
		return "hdr_end(host) -i " + v[1:]
	}
	return "hdr(host) -i " + v
}

func SNIMatcher(offloadSSL bool, host string) string {
	if offloadSSL {
		if strings.HasPrefix(host, "*") {
			return "ssl_fc_sni_end -i " + host[1:]
		}
		return "ssl_fc_sni -i " + host
	} else {
		if strings.HasPrefix(host, "*") {
			return "req_ssl_sni -i -m end " + host[1:]
		}
		return "req_ssl_sni -i " + host
	}
}

func BackendHash(value string, index int, mode string) string {
	if mode == "md5" {
		hash := md5.Sum([]byte(value))
		return base64.StdEncoding.EncodeToString(hash[:])
	} else if mode == "sha" {
		hash := sha512.Sum512([]byte(value))
		return base64.StdEncoding.EncodeToString(hash[:])
	} else if mode == "index" {
		return fmt.Sprintf("%d", index+1)
	}
	return value
}

var (
	funcMap = template.FuncMap{
		"acl_name":     ACLName,
		"header_name":  HeaderName,
		"host_acls":    HostACLs,
		"sni_matcher":  SNIMatcher,
		"backend_hash": BackendHash,
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
