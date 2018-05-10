package template

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/appscode/go/log"
	hpi "github.com/appscode/voyager/pkg/haproxy/api"
	"github.com/pkg/errors"
)

func RenderConfig(data hpi.TemplateData) (string, error) {
	data.Canonicalize()
	if err := data.IsValid(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err := haproxyTemplate.ExecuteTemplate(&buf, "haproxy.cfg", data)
	if err != nil {
		log.Error(err)
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

func CheckHAProxyConfig(cfg string) error {
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
