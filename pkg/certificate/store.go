package certificate

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path"

	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/appscode/voyager/client/clientset/versioned/typed/voyager/v1beta1/util"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/xenolf/lego/acme"
	core "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type CertStore struct {
	KubeClient    kubernetes.Interface
	VoyagerClient cs.Interface
	VaultClient   *vault.Client
}

func NewCertStore(kubeClient kubernetes.Interface, voyagerClient cs.Interface) (*CertStore, error) {
	store := &CertStore{
		KubeClient:    kubeClient,
		VoyagerClient: voyagerClient,
	}

	var err error
	if os.Getenv(vault.EnvVaultAddress) != "" {
		store.VaultClient, err = vault.NewClient(vault.DefaultConfig())
		if err != nil {
			return nil, err
		}
		if os.Getenv(vault.EnvVaultToken) == "" {
			token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
			if err != nil {
				return nil, err
			}
			store.VaultClient.SetToken(string(token))
		}
	}
	return store, nil
}

func (s *CertStore) Get(crd *api.Certificate) (pemCrt, pemKey []byte, err error) {
	if crd.Spec.Storage.Vault != nil {
		var secret *vault.Secret
		secret, err = s.VaultClient.Logical().Read(path.Join(crd.Spec.Storage.Vault.Prefix, crd.Namespace, crd.SecretName()))
		if err != nil {
			return
		}
		if secret == nil && err == nil {
			return nil, nil, nil
		}

		if data, found := secret.Data[core.TLSCertKey]; !found {
			err = errors.Errorf("secret %s/%s is missing tls.crt", crd.Namespace, crd.SecretName())
			return
		} else {
			pemCrt = []byte(data.(string))
		}
		if data, found := secret.Data[core.TLSPrivateKeyKey]; !found {
			err = errors.Errorf("secret %s/%s is missing tls.key", crd.Namespace, crd.SecretName())
			return
		} else {
			pemKey = []byte(data.(string))
		}
	} else {
		var secret *core.Secret
		secret, err = s.KubeClient.CoreV1().Secrets(crd.Namespace).Get(crd.SecretName(), metav1.GetOptions{})
		if k8serror.IsNotFound(err) {
			return nil, nil, nil
		}
		if err == nil {
			if data, found := secret.Data[core.TLSCertKey]; !found {
				err = errors.Errorf("secret %s/%s is missing tls.crt", crd.Namespace, crd.SecretName())
				return
			} else {
				pemCrt = data
			}
			if data, found := secret.Data[core.TLSPrivateKeyKey]; !found {
				err = errors.Errorf("secret %s/%s is missing tls.key", crd.Namespace, crd.SecretName())
				return
			} else {
				pemKey = data
			}
		}
	}
	return
}

func (s *CertStore) Save(crd *api.Certificate, cert *acme.CertificateResource) error {
	if crd.Spec.Storage.Vault != nil {
		data := map[string]interface{}{
			core.TLSCertKey:       string(cert.Certificate),
			core.TLSPrivateKeyKey: string(cert.PrivateKey),
		}
		_, err := s.VaultClient.Logical().Write(path.Join(crd.Spec.Storage.Vault.Prefix, crd.Namespace, crd.SecretName()), data)
		if err != nil {
			return err
		}
	} else {
		_, _, err := core_util.CreateOrPatchSecret(s.KubeClient,
			metav1.ObjectMeta{Namespace: crd.Namespace, Name: crd.SecretName()},
			func(in *core.Secret) *core.Secret {
				in.Type = core.SecretTypeTLS
				if in.Data == nil {
					in.Data = make(map[string][]byte)
				}
				in.Data[core.TLSCertKey] = cert.Certificate
				in.Data[core.TLSPrivateKeyKey] = cert.PrivateKey
				return in
			})
		if err != nil {
			return err
		}
	}

	// Decode cert
	pemBlock, _ := pem.Decode(cert.Certificate)
	crt, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errors.Errorf("failed to parse tls.crt for Certificate %s/%s. Reason: %s", crd.Namespace, crd.Name, err)
	}
	_, _, err = util.PatchCertificate(s.VoyagerClient.VoyagerV1beta1(), crd, func(in *api.Certificate) *api.Certificate {
		// Update certificate data to add Details Information
		t := metav1.Now()
		in.Status.LastIssuedCertificate = &api.CertificateDetails{
			SerialNumber:  crt.SerialNumber.String(),
			NotBefore:     metav1.NewTime(crt.NotBefore),
			NotAfter:      metav1.NewTime(crt.NotAfter),
			CertURL:       cert.CertURL,
			CertStableURL: cert.CertStableURL,
			AccountRef:    cert.AccountRef,
		}

		found := false
		for i := range in.Status.Conditions {
			if in.Status.Conditions[i].Type == api.CertificateIssued {
				in.Status.Conditions[i].LastUpdateTime = t
				found = true
			}
		}
		if !found {
			in.Status.Conditions = append(in.Status.Conditions, api.CertificateCondition{
				Type:           api.CertificateIssued,
				LastUpdateTime: t,
			})
		}
		return in
	})
	return err
}
