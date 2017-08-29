package util

import (
	"fmt"
	"io"
	"strings"

	"github.com/appscode/log"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func Exec(restClient rest.Interface, config *rest.Config, pod apiv1.Pod, cmd []string) string {
	req := restClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", pod.Spec.Containers[0].Name).
		Param("command", "sh").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "false")

	fmt.Println(req.URL())

	exec, err := remotecommand.NewExecutor(config, "POST", req.URL())
	if err != nil {
		log.Errorln(err)
		return ""
	}

	dw := &StringWriter{
		data: make([]byte, 0),
	}

	err = exec.Stream(remotecommand.StreamOptions{
		SupportedProtocols: remotecommandconsts.SupportedStreamingProtocols,
		Stdin:              newStringReader(cmd),
		Stdout:             dw,
		Stderr:             dw,
		Tty:                false,
	})
	if err != nil {
		log.Errorln(err)
		return ""
	}
	return dw.Output()
}

func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}

type StringWriter struct {
	data []byte
}

func (s *StringWriter) Flush() {
	s.data = make([]byte, 0)
}

func (s *StringWriter) Output() string {
	return string(s.data)
}

func (s *StringWriter) Write(b []byte) (int, error) {
	log.Infoln("$ ", string(b))
	s.data = append(s.data, b...)
	return len(b), nil
}
