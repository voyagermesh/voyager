package operator

import (
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op *Operator) MigrateCertificates() error {
	certs, err := op.ExtClient.Certificates(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cert := range certs.Items {
		err := op.EnsureCertificateMigration(&cert)
		if err != nil {
			return err
		}
	}
	return nil
}

func (op *Operator) EnsureCertificateMigration(cert *api.Certificate) error {
	migrated, migratedCert, secret := cert.Migrate()
	if migrated {
		if secret != nil {
			sec, err := op.KubeClient.CoreV1().Secrets(cert.Namespace).Get(secret.Name, metav1.GetOptions{})
			if err == nil {
				sec.Data = secret.Data
				_, err = op.KubeClient.CoreV1().Secrets(cert.Namespace).Update(sec)
				return err
			} else if errors.IsNotFound(err) {
				_, err = op.KubeClient.CoreV1().Secrets(cert.Namespace).Create(secret)
				if err != nil {
					return err
				}
			}
			return err
		}

		_, err := op.ExtClient.Certificates(cert.Namespace).Update(migratedCert)
		return err
	}
	return nil
}
