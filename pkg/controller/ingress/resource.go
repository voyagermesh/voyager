package ingress

import (
	"github.com/appscode/log"
	k8error "k8s.io/kubernetes/pkg/api/errors"
)

func (lbc *EngressController) IsExists() bool {
	lbc.parse()
	log.Infoln("Checking Ingress existence", lbc.Config.ObjectMeta)
	name := VoyagerPrefix + lbc.Config.Name
	if lbc.Options.LBType == LBHostPort {
		_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Get(name)
		if k8error.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.Extensions().Deployments(lbc.Config.Namespace).Get(name)
		if k8error.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(name)
	if k8error.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Get(name)
	if k8error.IsNotFound(err) {
		return false
	}
	return true
}
