package controller

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/appscode/go/log"
	apiv1 "k8s.io/api/core/v1"
)

var updateReceived, mountPerformed uint64

func incUpdateReceivedCounter() {
	atomic.AddUint64(&updateReceived, 1)
	log.Infoln("Update Received:", atomic.LoadUint64(&updateReceived))
}

func incMountCounter() {
	atomic.AddUint64(&mountPerformed, 1)
	log.Infoln("Mount Performed:", atomic.LoadUint64(&mountPerformed))
}

func namespace() string {
	if ns := os.Getenv("KUBE_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return apiv1.NamespaceDefault
}

func runCmd(path string) error {
	log.Infoln("calling boot file to execute")
	output, err := exec.Command("sh", "-c", path).CombinedOutput()
	msg := fmt.Sprintf("%v", string(output))
	log.Infoln("Output:\n", msg)
	if err != nil {
		log.Errorln("failed to run cmd")
		return fmt.Errorf("error restarting %v: %v", msg, err)
	}
	log.Infoln("boot file executed")
	return nil
}
