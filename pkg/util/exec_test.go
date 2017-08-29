package util

import (
	"fmt"
	"log"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func TestExec(t *testing.T) {
	pod := os.Getenv("EXEC_POD_NAME")
	if len(pod) > 0 {
		log.Println("Running Tests for pod", pod, "namespace", os.Getenv("EXEC_POD_NAMESPACE"), "container", os.Getenv("EXEC_CONTAINER_NAME"))
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			t.Error(err)
		}
		kubeClient := clientset.NewForConfigOrDie(config)
		output := Exec(
			kubeClient.CoreV1().RESTClient(),
			config,
			v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: pod, Namespace: os.Getenv("EXEC_POD_NAMESPACE")},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Name: os.Getenv("EXEC_CONTAINER_NAME")}}}},
			[]string{
				"cat /etc/hosts",
			},
		)
		fmt.Println(">>>", output)
	}
}
