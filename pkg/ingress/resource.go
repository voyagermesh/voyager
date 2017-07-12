package ingress

import (
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (lbc *Controller) IsExists() bool {
	log.Infoln("Checking Ingress existence", lbc.Ingress.ObjectMeta)
	if lbc.Ingress.LBType() == api.LBTypeHostPort {
		_, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	return true
}
