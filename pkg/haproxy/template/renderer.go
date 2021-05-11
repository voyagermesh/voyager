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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	hpi "voyagermesh.dev/voyager/pkg/haproxy/api"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

func RenderConfig(data hpi.TemplateData) (string, error) {
	data.Canonicalize()
	if err := data.IsValid(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err := haproxyTemplate.ExecuteTemplate(&buf, "haproxy.cfg", data)
	if err != nil {
		klog.Error(err)
		return "", err
	}
	lines := strings.Split(buf.String(), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n"), nil
}

func ValidateConfig(cfg string) error {
	tmpfile, err := ioutil.TempFile("", "haproxy-config-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name()) // clean up
	if _, err := tmpfile.Write([]byte(cfg)); err != nil {
		return err
	}
	if output, err := exec.Command("haproxy", "-c", "-f", tmpfile.Name()).CombinedOutput(); err != nil {
		return errors.Errorf("invalid haproxy configuration, reason: %s, output: %s", err, string(output))
	}
	return nil
}
