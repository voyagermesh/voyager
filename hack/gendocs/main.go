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

package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"voyagermesh.dev/voyager/pkg/cmds"

	"github.com/spf13/cobra/doc"
	"gomodules.xyz/runtime"
	"k8s.io/klog/v2"
)

var (
	tplFrontMatter = template.Must(template.New("index").Parse(`---
title: Reference
description: Voyager CLI Reference
menu:
  product_voyager_{{ "{{ .version }}" }}:
    identifier: reference
    name: Reference
    weight: 1000
menu_name: product_voyager_{{ "{{ .version }}" }}
---
`))

	_ = template.Must(tplFrontMatter.New("cmd").Parse(`---
title: {{ .Name }}
menu:
  product_voyager_{{ "{{ .version }}" }}:
    identifier: {{ .ID }}
    name: {{ .Name }}
    parent: reference
{{- if .RootCmd }}
    weight: 0
{{ end }}
product_name: voyager
menu_name: product_voyager_{{ "{{ .version }}" }}
section_menu_id: reference
{{- if .RootCmd }}
aliases:
  - /products/voyager/{{ "{{ .version }}" }}/reference/
{{ end }}
---
`))
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := cmds.NewCmdVoyager()
	dir := runtime.GOPath() + "/src/voyagermesh.dev/voyager/docs/reference"
	fmt.Printf("Generating cli markdown tree in: %v\n", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		klog.Fatalln(err)
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		klog.Fatalln(err)
	}

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		data := struct {
			ID      string
			Name    string
			RootCmd bool
		}{
			strings.Replace(base, "_", "-", -1),
			strings.Title(strings.Replace(base, "_", " ", -1)),
			!strings.ContainsRune(base, '_'),
		}
		var buf bytes.Buffer
		if err := tplFrontMatter.ExecuteTemplate(&buf, "cmd", data); err != nil {
			klog.Fatalln(err)
		}
		return buf.String()
	}

	linkHandler := func(name string) string {
		return "/docs/reference/" + name
	}
	err = doc.GenMarkdownTreeCustom(rootCmd, dir, filePrepender, linkHandler)
	if err != nil {
		klog.Fatalln(err)
	}
	index := filepath.Join(dir, "_index.md")
	f, err := os.OpenFile(index, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		klog.Fatalln(err)
	}
	err = tplFrontMatter.ExecuteTemplate(f, "index", struct{}{})
	if err != nil {
		klog.Fatalln(err)
	}
	if err := f.Close(); err != nil {
		klog.Fatalln(err)
	}
}
