package e2e

import (
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
)

func (i *IngressTestSuit) TestIngressEnsureTPR() error {
	var err error
	for it := 0; it < 10; it++ {
		log.Infoln(it, "Trying to get ingress.appscode.com")
		tpr, err := i.t.KubeClient.Extensions().ThirdPartyResources().Get("ingress.appscode.com")
		if err == nil {
			log.Infoln("Found tpr for ingress with name", tpr.Name)
			break
		}
		err = errors.New().WithCause(err).Internal()
		time.Sleep(time.Second * 5)
		continue
	}
	return err
}

func (i *IngressTestSuit) TestIngressCreate() error {
	return nil
}
