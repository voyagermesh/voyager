package e2e

import (
	"testing"
	"os/exec"
	"fmt"
)

func TestCmd(t *testing.T) {
	out, err := exec.Command("minikube","service", "hello-world", "--url").Output()
	fmt.Println(string(out), err)
}