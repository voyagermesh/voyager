package ingress

import (
	"github.com/appscode/log"
	k8error "k8s.io/kubernetes/pkg/api/errors"
)

func (lbc *EngressController) IsExists() bool {
	lbc.parse()
	log.Infoln("Checking Ingress existence", lbc.Config.ObjectMeta)
	if lbc.Options.LBType == LBDaemon {
		_, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
		if k8error.IsNotFound(err) {
			return false
		}
	} else {
		_, err := lbc.KubeClient.Core().ReplicationControllers(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
		if k8error.IsNotFound(err) {
			return false
		}
	}

	_, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if k8error.IsNotFound(err) {
		return false
	}

	_, err = lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if k8error.IsNotFound(err) {
		return false
	}
	return true
}
