package template

import (
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
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

func HostACLs(host string, port int, nodePort, redirectToPort int32, useNodePort bool) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		panic(fmt.Sprintf("found empty host for port %d", port))
	}

	var conditions []string

	if useNodePort && nodePort > 0 {
		conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, nodePort)))
	} else if !useNodePort && port > 0 {
		if port != 80 && port != 443 { // non standard http ports
			conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, port)))
		} else { // http or https
			conditions = append(conditions, hostMatcher(host))
			conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, port)))
		}
	}

	if port == 80 && redirectToPort > 0 {
		conditions = append(conditions, hostMatcher(fmt.Sprintf("%s:%d", host, redirectToPort)))
	}

	return conditions
}

func hostMatcher(v string) string {
	if strings.HasPrefix(v, "*") {
		return "hdr_end(host) -i " + v[1:]
	}
	return "hdr(host) -i " + v
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
