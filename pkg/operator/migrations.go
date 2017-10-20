package operator

import (
	v1u "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op *Operator) MigrateCertificates() error {
	certs, err := op.VoyagerClient.Certificates(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cert := range certs.Items {
		migrate, err := op.MigrateCertificate(&cert)
		if err != nil {
			return err
		}
		if migrate {
			_, err := op.VoyagerClient.Certificates(cert.Namespace).Update(&cert)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// https://github.com/appscode/voyager/issues/505
func (op *Operator) MigrateCertificate(cert *api.Certificate) (bool, error) {
	migrate := false
	if cert.Spec.Provider == "http" {
		cert.Spec.ChallengeProvider.HTTP = &api.HTTPChallengeProvider{
			Ingress: cert.Spec.HTTPProviderIngressReference,
		}
		migrate = true
	} else if len(cert.Spec.Provider) > 0 {
		cert.Spec.ChallengeProvider.DNS = &api.DNSChallengeProvider{
			Provider:             cert.Spec.Provider,
			CredentialSecretName: cert.Spec.ProviderCredentialSecretName,
		}
		migrate = true
	}
	if cert.Spec.Storage.Secret == nil && cert.Spec.Storage.Vault == nil {
		migrate = true
		cert.Spec.Storage = api.CertificateStorage{
			Secret: &apiv1.LocalObjectReference{},
		}
	}

	if cert.Spec.ACMEUserSecretName == "" {
		cert.Spec.ACMEUserSecretName = "acme-" + cert.Name
		migrate = true
	}

	if cert.Spec.Email != "" {
		_, err := v1u.CreateOrPatchSecret(op.KubeClient, metav1.ObjectMeta{
			Name:      cert.Spec.ACMEUserSecretName,
			Namespace: cert.Namespace,
		}, func(in *apiv1.Secret) *apiv1.Secret {
			if in.Data == nil {
				in.Data = make(map[string][]byte)
			}
			in.Data[api.ACMEUserEmail] = []byte(cert.Spec.Email)
			if len(cert.Spec.ACMEServerURL) != 0 {
				in.Data[api.ACMEServerURL] = []byte(cert.Spec.ACMEServerURL)
			}
			return in
		})
		if err != nil {
			return false, err
		}
	}

	// Setting deprecated values to zero value
	cert.Spec.Provider = ""
	cert.Spec.ProviderCredentialSecretName = ""
	cert.Spec.Email = ""
	cert.Spec.ACMEServerURL = ""
	cert.Spec.HTTPProviderIngressReference = api.LocalTypedReference{}

	cert.Status.CertificateObtained = false
	cert.Status.Message = ""
	cert.Status.ACMEUserSecretName = ""
	cert.Status.Details = nil

	return migrate, nil
}
