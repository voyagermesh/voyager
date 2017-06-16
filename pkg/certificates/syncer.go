package certificates

import (
	"time"
	"github.com/appscode/errors"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/benbjohnson/clock"
apiv1 "k8s.io/client-go/pkg/api/v1"
clientset "k8s.io/client-go/kubernetes"
)

type CertificateSyncer struct {
	CertificateController
	Time clock.Clock
}

func NewCertificateSyncer(c clientset.Interface, a acs.ExtensionInterface) *CertificateSyncer {
	return &CertificateSyncer{
		CertificateController: *NewController(c, a),
		Time: clock.New(),
	}
}

func (c *CertificateSyncer) RunSync() error {
	for {
		select {
		case <-c.Time.After(time.Hour * 24):
			certificates, err := c.ExtClient.Certificate(apiv1.NamespaceAll).List(apiv1.ListOptions{})
			if err != nil {
				return errors.FromErr(err).Err()
			}
			for _, cert := range certificates.Items {
				c.process(&cert)
			}
		}
	}
}
