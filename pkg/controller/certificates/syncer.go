package certificates

import (
	"time"

	"github.com/appscode/errors"
	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/benbjohnson/clock"
	"k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

type CertificateSyncer struct {
	CertificateController
	Time clock.Clock
}

func NewCertificateSyncer(c clientset.Interface, a acs.AppsCodeExtensionInterface) *CertificateSyncer {
	return &CertificateSyncer{
		CertificateController: *NewController(c, a),
		Time: clock.New(),
	}
}

func (c *CertificateSyncer) RunSync() error {
	for {
		select {
		case <-c.Time.After(time.Hour * 24):
			certificates, err := c.ACExtensionClient.Certificate(api.NamespaceAll).List(api.ListOptions{})
			if err != nil {
				return errors.New().WithCause(err).Internal()
			}
			for _, cert := range certificates.Items {
				c.process(&cert)
			}
		}
	}
}
