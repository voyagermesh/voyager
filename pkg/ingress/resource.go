package ingress

import (
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (lbc *Controller) IsExists() bool {
	log.Infoln("Checking Ingress existence", lbc.Resource.ObjectMeta)
	if lbc.Resource.LBType() == api.LBTypeHostPort {
		_, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.CoreV1().ConfigMaps(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	return true
}
