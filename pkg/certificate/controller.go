package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	v1u "github.com/appscode/kutil/core/v1"
	vu "github.com/appscode/kutil/voyager/v1beta1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	vault "github.com/hashicorp/vault/api"
	"github.com/xenolf/lego/acme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/cert"
)

type Controller struct {
	KubeClient clientset.Interface
	ExtClient  acs.VoyagerV1beta1Interface
	Opt        config.Options
	recorder   record.EventRecorder

	crd               *api.Certificate
	ChallengeProvider string
	UserSecret        *apiv1.Secret
	DNSCredentials    map[string][]byte
	curCert *x509.Certificate
	acmeUser   *ACMEUser
	acmeClient *acme.Client

	vaultClient *vault.Client
}

func NewController(kubeClient clientset.Interface, extClient acs.VoyagerV1beta1Interface, opt config.Options, tpr *api.Certificate) (*Controller, error) {
	ctrl := &Controller{
		KubeClient: kubeClient,
		ExtClient:  extClient,
		Opt:        opt,
		crd:        tpr,
		recorder:   eventer.NewEventRecorder(kubeClient, "voyager-operator"),
	}
	err := ctrl.crd.IsValid(ctrl.Opt.CloudProvider)
	if err != nil {
		return nil, err
	}

	ctrl.UserSecret, err = ctrl.KubeClient.CoreV1().Secrets(ctrl.crd.Namespace).Get(ctrl.crd.Spec.ACMEUserSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ctrl.acmeUser = &ACMEUser{}

	if email, ok := ctrl.UserSecret.Data[api.ACMEUserEmail]; !ok {
		return nil, fmt.Errorf("no acme user email is provided")
	} else {
		ctrl.acmeUser.Email = string(email)
	}

	if u, found := ctrl.UserSecret.Data[api.ACMEServerURL]; found {
		ctrl.acmeUser.ServerURL = string(u)
	} else {
		ctrl.acmeUser.ServerURL = LetsEncryptProdURL
	}

	if ctrl.crd.Spec.ChallengeProvider.HTTP != nil {
		ctrl.ChallengeProvider = "http"
		switch ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.APIVersion {
		case api.SchemeGroupVersion.String():
			var err error
			_, err = ctrl.ExtClient.Ingresses(ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
				Get(ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
		case "extensions/v1beta1":
			ing, err := ctrl.KubeClient.ExtensionsV1beta1().Ingresses(ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
				Get(ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			_, err = api.NewEngressFromIngress(ing)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("ingress API Schema unrecognized")
		}
	} else if ctrl.crd.Spec.ChallengeProvider.DNS != nil {
		ctrl.ChallengeProvider = ctrl.crd.Spec.ChallengeProvider.DNS.Provider
		if ctrl.crd.Spec.ChallengeProvider.DNS.CredentialSecretName != "" {
			dnsSecret, err := ctrl.KubeClient.CoreV1().Secrets(ctrl.crd.Namespace).Get(ctrl.crd.Spec.ChallengeProvider.DNS.CredentialSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			if _, ok := dnsSecret.Data[api.ACMEUserEmail]; !ok {
				return nil, fmt.Errorf("dns challenge provider credential %s not found", ctrl.crd.Spec.ChallengeProvider.DNS.CredentialSecretName)
			}
			ctrl.DNSCredentials = dnsSecret.Data
		}
	}

	if os.Getenv(vault.EnvVaultAddress) != "" {
		ctrl.vaultClient, err = vault.NewClient(vault.DefaultConfig())
		if err != nil {
			return nil, err
		}
		if os.Getenv(vault.EnvVaultToken) == "" {
			token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
			if err != nil {
				return nil, err
			}
			ctrl.vaultClient.SetToken(string(token))
		}
	} else {
		if ctrl.crd.Spec.Storage.Vault != nil {
			return nil, fmt.Errorf("certificate %s@%s uses vault but vault address is missing", tpr.Name, tpr.Namespace)
		}
	}

	if ctrl.crd.Spec.Storage.Secret != nil {
		if ctrl.crd.Spec.Storage.Secret.Name == "" {
			ctrl.crd.Spec.Storage.Secret.Name = "cert-" + tpr.Name
		}
	} else if ctrl.crd.Spec.Storage.Vault != nil {
		if ctrl.crd.Spec.Storage.Vault.Name == "" {
			ctrl.crd.Spec.Storage.Vault.Name = "cert-" + tpr.Name
		}
	}

	return ctrl, nil
}

func (c *Controller) Process() error {
	var err error
	c.curCert, _, err = c.load()
	if err != nil {
		return err
	}
	if c.curCert == nil {
		return c.create()
	}
	if c.crd.ShouldRenew(c.curCert) {
		return c.renew()
	}
	return nil
}

func (c *Controller) getACMEClient() error {
	var err error

	if data, ok := c.UserSecret.Data[api.ACMERegistrationData]; ok {
		var reg acme.RegistrationResource
		if err := json.Unmarshal(data, &reg); err == nil {
			c.acmeUser.Registration = &reg
		}
	}

	if data, ok := c.UserSecret.Data[api.ACMEUserPrivatekey]; ok {
		if key, err := cert.ParsePrivateKeyPEM(data); err == nil {
			if rsaKey, ok := key.(*rsa.PrivateKey); ok {
				c.acmeUser.Key = rsaKey
			}
		}
	}
	registered := c.acmeUser.Registration != nil && c.acmeUser.Key != nil

	if c.acmeUser.Key == nil {
		log.Infoln("No ACME user found, registering a new ACME user")
		userKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errors.FromErr(err).WithMessage("Failed to generate Key for New Acme User")
		}
		c.acmeUser.Key = userKey
	}

	c.acmeClient, err = c.newACMEClient()
	if err != nil {
		return err
	}

	if !registered {
		registration, err := c.acmeClient.Register()
		if err != nil {
			return fmt.Errorf("failed to register user %s. Reason: %s", c.acmeUser.Email, err)
		}
		if err := c.acmeClient.AgreeToTOS(); err != nil {
			return fmt.Errorf("failed to register user %s. Reason: %s", c.acmeUser.Email, err)
		}
		c.UserSecret, err = v1u.PatchSecret(c.KubeClient, c.UserSecret, func(in *apiv1.Secret) *apiv1.Secret {
			if in.Data == nil {
				in.Data = make(map[string][]byte)
			}
			in.Data[api.ACMEUserPrivatekey] = cert.EncodePrivateKeyPEM(c.acmeUser.Key.(*rsa.PrivateKey))
			regBytes, _ := json.Marshal(registration)
			in.Data[api.ACMERegistrationData] = regBytes
			return in
		})
		return err
	}
	return nil
}

func (c *Controller) create() error {
	if err := c.getACMEClient(); err != nil {
		return err
	}
	if c.ChallengeProvider == "http" {
		if err := c.processHTTPCertificate(); err != nil {
			return err
		}
	}
	cert, errs := c.acmeClient.ObtainCertificate(c.crd.Spec.Domains, true, nil, true)
	if len(errs) > 0 {
		causes := make([]string, 0, len(errs))
		for k, v := range errs {
			causes = append(causes, k+": "+v.Error())
		}
		return fmt.Errorf("failed to create certificate. Reason: %s", strings.Join(causes, ", "))
	}
	return c.save(cert)
}

func (c *Controller) renew() error {
	if err := c.getACMEClient(); err != nil {
		return err
	}
	if c.ChallengeProvider == "http" {
		if err := c.processHTTPCertificate(); err != nil {
			return err
		}
	}
	acmeCert := acme.CertificateResource{
		CertURL:       c.crd.Status.LastIssuedCertificate.CertURL,
		CertStableURL: c.crd.Status.LastIssuedCertificate.CertStableURL,
		AccountRef:    c.crd.Status.LastIssuedCertificate.AccountRef,
		Certificate:   c.curCert.Raw,
		PrivateKey:    nil, // issue new private key,
	}
	cert, err := c.acmeClient.RenewCertificate(acmeCert, true, true)
	if err != nil {
		return err
	}
	return c.save(cert)
}

func (c *Controller) load() (crt *x509.Certificate, key *rsa.PrivateKey, err error) {
	var pemCrt, pemKey []byte

	if c.crd.Spec.Storage.Secret != nil {
		var secret *apiv1.Secret
		secret, err = c.KubeClient.CoreV1().Secrets(c.crd.Namespace).Get(c.crd.Spec.Storage.Secret.Name, metav1.GetOptions{})
		if err == nil {
			if data, found := secret.Data["tls.crt"]; !found {
				err = fmt.Errorf("secret %s@%s is missing tls.crt", c.crd.Spec.Storage.Secret.Name, c.crd.Namespace)
				return
			} else {
				pemCrt = data
			}
			if data, found := secret.Data["tls.key"]; !found {
				err = fmt.Errorf("secret %s@%s is missing tls.key", c.crd.Spec.Storage.Secret.Name, c.crd.Namespace)
				return
			} else {
				pemKey = data
			}
		}
	} else if c.crd.Spec.Storage.Vault != nil {
		var secret *vault.Secret
		secret, err = c.vaultClient.Logical().Read(path.Join(c.crd.Spec.Storage.Vault.Prefix, c.crd.Namespace, c.crd.Spec.Storage.Vault.Name))
		if err != nil {
			return
		}
		if data, found := secret.Data["tls.crt"]; !found {
			err = fmt.Errorf("secret %s@%s is missing tls.crt", c.crd.Spec.Storage.Secret.Name, c.crd.Namespace)
			return
		} else {
			pemCrt = []byte(data.(string))
		}
		if data, found := secret.Data["tls.key"]; !found {
			err = fmt.Errorf("secret %s@%s is missing tls.key", c.crd.Spec.Storage.Secret.Name, c.crd.Namespace)
			return
		} else {
			pemKey = []byte(data.(string))
		}
	}

	if len(pemCrt) > 0 {
		var certs []*x509.Certificate
		certs, err = cert.ParseCertsPEM(pemCrt)
		if err != nil {
			err = fmt.Errorf("secret %s@%s contains bad certificate. Reason: %s", c.crd.Spec.Storage.Secret.Name, c.crd.Namespace, err)
			return
		}
		crt = certs[0]
	}
	if len(pemKey) > 0 {
		var ki interface{}
		ki, err = cert.ParsePrivateKeyPEM(pemKey)
		if err != nil {
			return
		}
		if rsaKey, ok := ki.(*rsa.PrivateKey); ok {
			key = rsaKey
		} else {
			err = fmt.Errorf("key datya is not rsa private key")
		}
		return
	}
	return
}

func (c *Controller) save(cert acme.CertificateResource) error {
	if c.crd.Spec.Storage.Secret != nil {
		_, err := v1u.CreateOrPatchSecret(c.KubeClient,
			metav1.ObjectMeta{Namespace: c.crd.Namespace, Name: c.crd.Spec.Storage.Secret.Name},
			func(in *apiv1.Secret) *apiv1.Secret {
				in.Type = apiv1.SecretTypeTLS
				if in.Data == nil {
					in.Data = make(map[string][]byte)
				}
				in.Data["tls.crt"] = cert.Certificate
				in.Data["tls.key"] = cert.PrivateKey
				return in
			})
		return err
	} else if c.crd.Spec.Storage.Vault != nil {
		data := map[string]interface{}{
			"tls.crt": string(cert.Certificate),
			"tls.key": string(cert.PrivateKey),
		}
		_, err := c.vaultClient.Logical().Write(path.Join(c.crd.Spec.Storage.Vault.Prefix, c.crd.Namespace, c.crd.Spec.Storage.Vault.Name), data)
		return err
	}

	// Decode cert
	pemBlock, _ := pem.Decode(cert.Certificate)
	crt, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errors.FromErr(err).WithMessage("Error decoding x509 encoded certificate").Err()
	}
	_, err = vu.PatchCertificate(c.ExtClient, c.crd, func(in *api.Certificate) *api.Certificate {
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

func (c *Controller) processHTTPCertificate() error {
	switch c.crd.Spec.ChallengeProvider.HTTP.Ingress.APIVersion {
	case api.SchemeGroupVersion.String():
		i, err := c.ExtClient.Ingresses(c.crd.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
			Get(c.crd.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}
		for _, rules := range i.Spec.Rules {
			if rules.HTTP != nil {
				for _, path := range rules.HTTP.Paths {
					if path.Path == providers.URLPrefix {
						// Rule found no update required
						return nil
					}
				}
			}
		}

		rule := api.IngressRule{
			IngressRuleValue: api.IngressRuleValue{
				HTTP: &api.HTTPIngressRuleValue{
					Paths: []api.HTTPIngressPath{
						{
							Path: providers.URLPrefix,
							Backend: api.HTTPIngressBackend{
								IngressBackend: api.IngressBackend{
									ServiceName: c.Opt.OperatorService + "." + c.Opt.OperatorNamespace,
									ServicePort: intstr.FromInt(c.Opt.HTTPChallengePort),
								},
							},
						},
					},
				},
			},
		}
		i.Spec.Rules = append(i.Spec.Rules, rule)

		_, err = c.ExtClient.Ingresses(c.crd.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
	case "extensions/v1beta1":
		i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.crd.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
			Get(c.crd.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		for _, rules := range i.Spec.Rules {
			if rules.HTTP != nil {
				for _, path := range rules.HTTP.Paths {
					if path.Path == providers.URLPrefix {
						// Rule found no update required
						return nil
					}
				}
			}
		}

		rule := extensions.IngressRule{
			IngressRuleValue: extensions.IngressRuleValue{
				HTTP: &extensions.HTTPIngressRuleValue{
					Paths: []extensions.HTTPIngressPath{
						{
							Path: providers.URLPrefix,
							Backend: extensions.IngressBackend{
								ServiceName: c.Opt.OperatorService + "." + c.Opt.OperatorNamespace,
								ServicePort: intstr.FromInt(c.Opt.HTTPChallengePort),
							},
						},
					},
				},
			},
		}
		i.Spec.Rules = append(i.Spec.Rules, rule)

		_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.crd.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
	default:
		return errors.New("HTTP Certificate resolver do not have any ingress reference or invalid ingress reference").Err()
	}
	return nil
}
