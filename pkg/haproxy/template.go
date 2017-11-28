package haproxy

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
	v = strings.Replace(v, "/", "_", -1)
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

func HostName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == `*` {
		return ""
	}
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
		"host_name":    HostName,
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
