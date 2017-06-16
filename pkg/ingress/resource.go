package ingress

import (
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	kerr "k8s.io/apimachinery/pkg/api/errors"
)

func (lbc *EngressController) IsExists() bool {
	log.Infoln("Checking Ingress existence", lbc.Resource.ObjectMeta)
	if lbc.Resource.LBType() == api.LBTypeHostPort || lbc.Resource.LBType() == api.LBTypeDaemon {
		_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
		if kerr.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
		if kerr.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
	if kerr.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName())
	if kerr.IsNotFound(err) {
		return false
	}
	return true
}
