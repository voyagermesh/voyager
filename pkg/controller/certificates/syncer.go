package certificates

import (
	"time"

	"github.com/appscode/errors"
	acs "github.com/appscode/k8s-addons/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

type CertificateSyncer struct {
	CertificateController
}

func NewCertificateSyncer(c clientset.Interface, a acs.AppsCodeExtensionInterface) *CertificateSyncer {
	return &CertificateSyncer{
		CertificateController: *NewController(c, a),
	}
}

func (c *CertificateSyncer) RunSync() error {
	for {
		select {
		case <-time.After(time.Hour * 24):
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
