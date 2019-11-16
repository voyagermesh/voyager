/*
Copyright The Voyager Authors.

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

package controller

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/golang/glog"
	ps "github.com/mitchellh/go-ps"
	"github.com/pkg/errors"
)

const (
	haproxyConfig = "/etc/haproxy/haproxy.cfg"
	haproxyPID    = "/var/run/haproxy.pid"
	haproxySocket = "/var/run/haproxy.sock"
)

func getHAProxyPid() (int, error) {
	file, err := os.Open(haproxyPID)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var pid int
	_, err = fmt.Fscan(file, &pid)
	return pid, err
}

func checkHAProxyDaemon() (int, error) {
	pid, err := getHAProxyPid()
	if err != nil {
		return 0, errors.Wrap(err, "error reading haproxy.pid file")
	}

	if process, err := ps.FindProcess(pid); err != nil {
		return 0, errors.Wrap(err, "failed to get haproxy daemon process")
	} else if process == nil {
		return 0, errors.Errorf("haproxy daemon not running (pid %d)", pid)
	}

	glog.Infof("haproxy daemon running (pid %d)", pid)
	return pid, nil
}

func checkHAProxyConfig() error {
	glog.Info("Checking haproxy config...")
	output, err := exec.Command("haproxy", "-c", "-f", haproxyConfig).CombinedOutput()
	if err != nil {
		return errors.Errorf("haproxy-check failed, reason: %s %s", string(output), err)
	}
	glog.Infof("haproxy-check: %s", string(output))
	return nil
}

func startHAProxy() error {
	if err := checkHAProxyConfig(); err != nil {
		return err
	}
	glog.Info("Starting haproxy...")

	output, err := exec.Command("haproxy", "-f", haproxyConfig, "-p", haproxyPID).CombinedOutput()
	if err != nil {
		return errors.Errorf("failed to start haproxy, reason: %s %s", string(output), err)
	}

	glog.Infof("haproxy started: %s", string(output))
	return nil
}

func reloadHAProxy(pid int) error {
	if err := checkHAProxyConfig(); err != nil {
		return err
	}
	glog.Info("Reloading haproxy...")

	output, err := exec.Command(
		"haproxy",
		"-f", haproxyConfig,
		"-p", haproxyPID,
		"-x", haproxySocket,
		"-sf", strconv.Itoa(pid),
	).CombinedOutput()
	if err != nil {
		return errors.Errorf("failed to reload haproxy, reason: %s %s", string(output), err)
	}

	glog.Infof("haproxy reloaded: %s", string(output))
	return nil
}

// reload if old haproxy daemon exists, otherwise start
func startOrReloadHaproxy() error {
	if pid, err := checkHAProxyDaemon(); err != nil {
		return startHAProxy()
	} else {
		return reloadHAProxy(pid)
	}
}
