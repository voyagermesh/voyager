package controller

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/golang/glog"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/errors"
)

const (
	HAPROXY_CONFIG = "/etc/haproxy/haproxy.cfg"
	HAPROXY_PID    = "/var/run/haproxy.pid"
	HAPROXY_SOCK   = "/var/run/haproxy.sock"
)

func getHAProxyPid() (int, error) {
	file, err := os.Open(HAPROXY_PID)
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
		return 0, errors.Errorf("error reading haproxy.pid file, reason %s", err)
	}

	if process, err := ps.FindProcess(pid); err != nil {
		return 0, errors.Errorf("failed to get haproxy daemon process, reason %s", err)
	} else if process == nil {
		return 0, errors.Errorf("haproxy daemon not running (pid %d)", pid)
	}

	glog.Infof("haproxy daemon running (pid %d)", pid)
	return pid, nil
}

func checkHAProxyConfig() error {
	glog.Info("Checking haproxy config...")
	output, err := exec.Command("haproxy", "-c", "-f", HAPROXY_CONFIG).CombinedOutput()
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

	output, err := exec.Command("haproxy", "-f", HAPROXY_CONFIG, "-p", HAPROXY_PID).CombinedOutput()
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
		"-f", HAPROXY_CONFIG,
		"-p", HAPROXY_PID,
		"-x", HAPROXY_SOCK,
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
