package ingress

import (
	"github.com/appscode/log"
	k8error "k8s.io/kubernetes/pkg/api/errors"
)

func (lbc *EngressController) IsExists() bool {
	lbc.parse()
	log.Infoln("Checking Ingress existence", lbc.Resource.ObjectMeta)
	name := VoyagerPrefix + lbc.Resource.Name
	if lbc.Options.LBType == LBTypeHostPort || lbc.Options.LBType == LBTypeDaemon {
		_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Get(name)
		if k8error.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(name)
		if k8error.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(name)
	if k8error.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Get(name)
	if k8error.IsNotFound(err) {
		return false
	}
	return true
}
