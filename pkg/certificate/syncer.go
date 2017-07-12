package certificate

import (
	"time"

	"github.com/appscode/log"
	tcs "github.com/appscode/voyager/client/clientset"
	"github.com/benbjohnson/clock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func CheckCertificates(kubeClient clientset.Interface, extClient tcs.ExtensionInterface) {
	Time := clock.New()
	for {
		select {
		case <-Time.After(time.Hour * 24):
			result, err := extClient.Certificates(apiv1.NamespaceAll).List(metav1.ListOptions{})
			if err != nil {
				log.Error(err)
				continue
			}
			for i := range result.Items {
				err = NewController(kubeClient, extClient, &result.Items[i]).Process()
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}
