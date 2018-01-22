package template

import (
	"bytes"
	"strings"

	"github.com/appscode/go/log"
	hpi "github.com/appscode/voyager/pkg/haproxy/api"
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
