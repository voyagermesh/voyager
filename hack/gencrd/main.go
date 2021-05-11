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
	"io/ioutil"
	"os"
	"path/filepath"

	"voyagermesh.dev/voyager/apis/voyager/install"
	v1beta1 "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	"github.com/go-openapi/spec"
	gort "gomodules.xyz/runtime"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/common"
	"kmodules.xyz/client-go/openapi"
)

func generateSwaggerJson() {
	var (
		Scheme = runtime.NewScheme()
		Codecs = serializer.NewCodecFactory(Scheme)
	)

	install.Install(Scheme)

	apispec, err := openapi.RenderOpenAPISpec(openapi.Config{
		Scheme: Scheme,
		Codecs: Codecs,
		Info: spec.InfoProps{
			Title:   "Voyager",
			Version: "v13.0.0-beta.1",
			Contact: &spec.ContactInfo{
				Name:  "AppsCode Inc.",
				URL:   "https://appscode.com",
				Email: "hello@appscode.com",
			},
			License: &spec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},
		OpenAPIDefinitions: []common.GetOpenAPIDefinitions{
			v1beta1.GetOpenAPIDefinitions,
		},
		Resources: []openapi.TypeInfo{
			{GroupVersion: v1beta1.SchemeGroupVersion, Resource: v1beta1.ResourceIngresses, Kind: v1beta1.ResourceKindIngress, NamespaceScoped: true},
		},
	})
	if err != nil {
		klog.Fatal(err)
	}

	filename := gort.GOPath() + "/src/voyagermesh.dev/voyager/openapi/swagger.json"
	err = os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		klog.Fatal(err)
	}
	err = ioutil.WriteFile(filename, []byte(apispec), 0644)
	if err != nil {
		klog.Fatal(err)
	}
}

func main() {
	generateSwaggerJson()
}
