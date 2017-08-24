package certificate

import (
	"time"

	"github.com/appscode/log"
	tcs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/benbjohnson/clock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

func CheckCertificates(config *rest.Config, kubeClient clientset.Interface, extClient tcs.ExtensionInterface, opt config.Options) {
	Time := clock.New()
	for {
		select {
		case <-Time.After(time.Minute * 5):
			result, err := extClient.Certificates(apiv1.NamespaceAll).List(metav1.ListOptions{})
			if err != nil {
				log.Error(err)
				continue
			}
			for i := range result.Items {
				err = NewController(config, kubeClient, extClient, opt, &result.Items[i]).Process()
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}
