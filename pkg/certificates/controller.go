package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"sync"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/certificates/providers"
	"github.com/xenolf/lego/acme"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	defaultCertPrefix       = "cert-"
	defaultUserSecretPrefix = "acme-"

	certificateAnnotationKeyEnabled                      = "certificate.appscode.com/enabled"
	certificateAnnotationKeyName                         = "certificate.appscode.com/name"
	certificateAnnotationKeyProvider                     = "certificate.appscode.com/provider"
	certificateAnnotationKeyEmail                        = "certificate.appscode.com/email"
	certificateAnnotationKeyProviderCredentialSecretName = "certificate.appscode.com/provider-secret"
	certificateAnnotationKeyACMEUserSecretName           = "certificate.appscode.com/user-secret"
	certificateAnnotationKeyACMEServerURL                = "certificate.appscode.com/server-url"
)

type Controller struct {
	// kubernetes clients
	KubeClient clientset.Interface
	ExtClient  acs.ExtensionInterface
}

type internalCertificateOptions struct {
	*Controller
	certificate       *api.Certificate
	acmeCert          ACMECertData
	parsedCertificate *x509.Certificate
	sync.Mutex

	acmeClientConfig *ACMEConfig
	acmeClient       *ACMEClient

	userSecretName string
}

func NewController(c clientset.Interface, a acs.ExtensionInterface) *Controller {
	return &Controller{
		KubeClient: c,
		ExtClient:  a,
	}
}

func (c *Controller) newInternalCertificateOptions(cert *api.Certificate) *internalCertificateOptions {
	return &internalCertificateOptions{
		Controller:  c,
		certificate: cert,
	}
}

func (c *Controller) HandleCertificate(cert *api.Certificate) error {
	return c.process(c.newInternalCertificateOptions(cert))
}

func (c *Controller) HandleIngress(ingress *api.Ingress) error {
	if ingress.Annotations != nil {
		if val, ok := ingress.Annotations[certificateAnnotationKeyEnabled]; ok && val == "true" {
			certificateName := ingress.Annotations[certificateAnnotationKeyName]
			// Check if a certificate already exists.
			certificate, err := c.ExtClient.Certificates(ingress.Namespace).Get(certificateName)
			if err == nil {
				// Certificate exists mount it.
				return nil
			}
			if kerr.IsNotFound(err) || !certificate.Status.CertificateObtained {
				newCertificate := &api.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      certificateName,
						Namespace: ingress.Namespace,
					},
					Spec: api.CertificateSpec{
						Provider: ingress.Annotations[certificateAnnotationKeyProvider],
						Email:    ingress.Annotations[certificateAnnotationKeyEmail],
						ProviderCredentialSecretName: ingress.Annotations[certificateAnnotationKeyProviderCredentialSecretName],
						HTTPProviderIngressReference: apiv1.ObjectReference{
							Kind:            "Ingress",
							Name:            ingress.Name,
							Namespace:       ingress.Namespace,
							ResourceVersion: ingress.ResourceVersion,
							UID:             ingress.UID,
						},
						ACMEUserSecretName: ingress.Annotations[certificateAnnotationKeyACMEUserSecretName],
						ACMEServerURL:      ingress.Annotations[certificateAnnotationKeyACMEServerURL],
					},
				}
				if v, ok := ingress.Annotations[api.APISchema]; ok {
					if v == api.APISchemaIngress {
						newCertificate.Spec.HTTPProviderIngressReference.APIVersion = api.APISchemaIngress
					} else {
						newCertificate.Spec.HTTPProviderIngressReference.APIVersion = api.APISchemaEngress
					}
				}
				for _, rule := range ingress.Spec.Rules {
					found := false
					for _, tls := range ingress.Spec.TLS {
						if strings.Contains(tls.Hosts, rule.Host) {
							found = true
						}
					}
					if !found {
						newCertificate.Spec.Domains = append(newCertificate.Spec.Domains, rule.Host)
					}
				}
				_, err := c.ExtClient.Certificates(newCertificate.Namespace).Create(newCertificate)
				if err != nil {
					errors.FromErr(err).Err()
				}
			}
		}
	}
	return nil
}

func (c *Controller) process(opt *internalCertificateOptions) error {
	opt.acmeClientConfig = &ACMEConfig{
		Provider:      opt.certificate.Spec.Provider,
		ACMEServerUrl: opt.certificate.Spec.ACMEServerURL,
	}

	// Check if a cert already exists for this Certificate Instance
	secret, err := c.KubeClient.CoreV1().Secrets(opt.certificate.Namespace).Get(defaultCertPrefix+opt.certificate.Name, metav1.GetOptions{})
	if err == nil {
		var err error
		opt.acmeCert, err = NewACMECertDataFromSecret(secret)
		if err != nil {
			return errors.FromErr(err).WithMessage("Error decoding acme certificate").Err()
		}

		// Decode cert
		pemBlock, _ := pem.Decode(opt.acmeCert.Cert)
		opt.parsedCertificate, err = x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			return errors.FromErr(err).WithMessage("Error decoding x509 encoded certificate").Err()
		}
		if !opt.parsedCertificate.NotAfter.After(time.Now().Add(time.Hour * 24 * 7)) {
			log.Infoln("certificate is expiring in 7 days, attempting renew")
			opt.renew()
		}

		if !opt.acmeCert.EqualDomains(opt.parsedCertificate) {
			opt.renew()
		}
	}

	if kerr.IsNotFound(err) || !opt.certificate.Status.CertificateObtained {
		// Certificate Not found as secret. We must create it now.
		opt.create()
	}
	return nil
}

func (c *internalCertificateOptions) create() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.FromErr(err).Err()
	}
	if c.certificate.Spec.Provider == "http" {
		done := make(chan struct{})
		defer func() {
			done <- struct{}{}
			close(done)
		}()
		if err := c.processHTTPCertificate(done); err != nil {
			return err
		}
	}
	cert, errs := c.acmeClient.ObtainCertificate(c.certificate.Spec.Domains, true, nil, true)
	for k, v := range errs {
		log.Errorf("Error occurred for %s, reason %s", k, v.Error())
	}
	if len(cert.PrivateKey) > 0 {
		return c.save(cert)
	}
	return nil
}

func (c *internalCertificateOptions) renew() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.FromErr(err).Err()
	}

	if c.certificate.Spec.Provider == "http" {
		done := make(chan struct{})
		defer func() {
			done <- struct{}{}
			close(done)
		}()
		if err := c.processHTTPCertificate(done); err != nil {
			return err
		}
	}
	acmeCert := acme.CertificateResource{
		Domain:        c.certificate.Status.Details.Domain,
		CertURL:       c.certificate.Status.Details.CertURL,
		CertStableURL: c.certificate.Status.Details.CertStableURL,
		AccountRef:    c.certificate.Status.Details.AccountRef,
		Certificate:   c.acmeCert.Cert,
		PrivateKey:    c.acmeCert.PrivateKey,
	}
	cert, err := c.acmeClient.RenewCertificate(acmeCert, true, true)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return c.update(cert)
}

func (c *internalCertificateOptions) ensureACMEClient() error {
	var acmeUserInfo *ACMEUserData
	acmeUserRegistered := false
	log.Infoln("trying to retrive acmeUser data")

	var userSecret *apiv1.Secret
	err := errors.New("Setting error Not found").Err()
	if c.certificate.Spec.ACMEUserSecretName != "" {
		// ACMEUser secret name is provided.
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Get(c.certificate.Spec.ACMEUserSecretName, metav1.GetOptions{})
	}
	if err != nil && c.certificate.Status.ACMEUserSecretName != "" {
		// There is a error getting the secret, try the secret name from status, if this was a update request
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Get(c.certificate.Status.ACMEUserSecretName, metav1.GetOptions{})
	}
	if err != nil {
		// Trying to find an secret with the same name of Certificates
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Get(defaultUserSecretPrefix+c.certificate.Name, metav1.GetOptions{})
		if err == nil {
			if _, ok := userSecret.Annotations[certificateKey+"/user-info"]; !ok {
				err = errors.Newf("No %s annotaion key", certificateKey+"/user-info").Err()
			}
		}
	}
	// No error that means we successfully got an userSecret
	if err == nil {
		c.userSecretName = userSecret.Name
		c.acmeClientConfig.UserDataMap = userSecret.Data
		if userInfo, ok := userSecret.Data["user-info"]; ok {
			acmeUserInfo = &ACMEUserData{}
			log.Info("ACMEUserInfo data found is secret", userSecret.Name)
			userError := json.Unmarshal(userInfo, acmeUserInfo)
			if userError == nil {
				acmeUserRegistered = true
			}
		}
	}

	if !acmeUserRegistered {
		log.Errorln("No ACME user found, Generate a new ACME user")
		userKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errors.FromErr(err).WithMessage("Failed to generate Key for New Acme User")
		}
		acmeUserInfo = &ACMEUserData{
			Email: c.certificate.Spec.Email,
			Key: pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(userKey),
			}),
		}
	}

	c.acmeClientConfig.UserData = acmeUserInfo
	// Initiate ACME Client for config.
	if err := c.loadProviderCredential(); err != nil {
		return errors.FromErr(err).Err()
	}

	log.V(9).Infoln("Getting NewACMECLient with config", c.acmeClientConfig)
	acmeClient, err := NewACMEClient(c.acmeClientConfig)
	if err != nil {
		return errors.FromErr(err).WithMessage("Failed to create acme client").Err()
	}
	c.acmeClient = acmeClient

	if !acmeUserRegistered {
		return c.registerACMEUser(acmeClient)
	}
	return nil
}

func (c *internalCertificateOptions) registerACMEUser(acmeClient *ACMEClient) error {
	log.Debugln("ACME user not registered, registering new ACME user")
	registration, err := acmeClient.Register()
	if err != nil {
		return errors.FromErr(err).WithMessage("Failed to registering user for new domain").Err()
	}
	c.acmeClientConfig.UserData.Registration = registration
	if err := acmeClient.AgreeToTOS(); err != nil {
		return errors.FromErr(err).WithMessage("Failed to registering user for new domain").Err()
	}

	// Acme User registered Create The acmeUserSecret
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.certificate.Spec.ACMEUserSecretName,
			Namespace: c.certificate.Namespace,
			Labels: map[string]string{
				certificateKey + "/user-info": "true",
				certificateKey + "/cert-name": c.certificate.Name,
			},
			Annotations: map[string]string{
				certificateKey + "/user-info": "true",
				certificateKey + "/cert-name": c.certificate.Name,
			},
		},
		Data: map[string][]byte{
			"user-info": c.acmeClientConfig.UserData.Json(),
		},
		Type: "certificate.appscode.com/acme-user-info",
	}
	if secret.Name == "" {
		secret.Name = defaultUserSecretPrefix + c.certificate.Name
	}
	c.userSecretName = secret.Name
	log.Debugln("User Registered saving User Informations with Secret name", c.userSecretName)
	_, err = c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Create(secret)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *internalCertificateOptions) loadProviderCredential() error {
	cred, err := c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Get(c.certificate.Spec.ProviderCredentialSecretName, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.acmeClientConfig.ProviderCredentials = cred.Data
	return nil
}

func (c *internalCertificateOptions) save(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.certificate.Name, c.certificate.Namespace)
	_, err := c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Create(secret)
	if err != nil {
		errors.FromErr(err).Err()
	}

	k8sCert, err := c.ExtClient.Certificates(c.certificate.Namespace).Get(c.certificate.Name)
	if err != nil {
		log.Errorln("failed to load cert object,", err)
	}

	// Update certificate data to add Details Information
	t := metav1.Now()
	k8sCert.Status = api.CertificateStatus{
		CertificateObtained: true,
		CreationTime:        &t,
		ACMEUserSecretName:  c.userSecretName,
		Details: api.ACMECertificateDetails{
			Domain:        cert.Domain,
			CertURL:       cert.CertURL,
			CertStableURL: cert.CertStableURL,
			AccountRef:    cert.AccountRef,
		},
	}
	c.ExtClient.Certificates(c.certificate.Namespace).Update(k8sCert)
	return nil
}

func (c *internalCertificateOptions) update(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.certificate.Name, c.certificate.Namespace)
	oldSecret, err := c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	oldSecret.Data = secret.Data
	_, err = c.KubeClient.CoreV1().Secrets(c.certificate.Namespace).Update(oldSecret)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *internalCertificateOptions) processHTTPCertificate(revert chan struct{}) error {
	c.acmeClient.HTTPProviderLock.Lock()
	defer c.acmeClient.HTTPProviderLock.Unlock()

	switch c.certificate.Spec.HTTPProviderIngressReference.APIVersion {
	case "appscode.com/v1":
		revertRequired := false
		i, err := c.ExtClient.Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
			Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		// make a copy of previous spec.
		prevSpecs := i.Spec
		for _, host := range c.certificate.Spec.Domains {
			rule := api.ExtendedIngressRule{
				Host: host,
				ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
					HTTP: &api.HTTPExtendedIngressRuleValue{
						Paths: []api.HTTPExtendedIngressPath{
							{
								Path: providers.URLPrefix,
								Backend: api.ExtendedIngressBackend{
									ServiceName: "kubed.kube-system",
									ServicePort: intstr.FromInt(8765),
								},
							},
						},
					},
				},
			}

			i.Spec.Rules = append(i.Spec.Rules, rule)
		}
		_, err = c.ExtClient.Ingresses(c.certificate.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.ExtClient.Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
						Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, api.ExtendedIngressTLS{
							Hosts:      c.certificate.Spec.Domains,
							SecretName: defaultCertPrefix + c.certificate.Name,
						})
						c.ExtClient.Ingresses(c.certificate.Namespace).Update(i)
					}
				}
				return
			}
		}()
	case "extensions/v1beta1":
		revertRequired := false
		i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).Get(c.certificate.Spec.HTTPProviderIngressReference.Name, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}
		// make a copy of previous spec.
		prevSpecs := i.Spec
		for _, host := range c.certificate.Spec.Domains {
			rule := extensions.IngressRule{
				Host: host,
				IngressRuleValue: extensions.IngressRuleValue{
					HTTP: &extensions.HTTPIngressRuleValue{
						Paths: []extensions.HTTPIngressPath{
							{
								Path: providers.URLPrefix,
								Backend: extensions.IngressBackend{
									ServiceName: "kubed.kube-system",
									ServicePort: intstr.FromInt(8765),
								},
							},
						},
					},
				},
			}

			i.Spec.Rules = append(i.Spec.Rules, rule)
		}
		_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.certificate.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).Get(c.certificate.Spec.HTTPProviderIngressReference.Name, metav1.GetOptions{})
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, extensions.IngressTLS{
							Hosts:      c.certificate.Spec.Domains,
							SecretName: defaultCertPrefix + c.certificate.Name,
						})
						c.KubeClient.ExtensionsV1beta1().Ingresses(c.certificate.Namespace).Update(i)
					}
				}
				return
			}
		}()
	default:
		return errors.New("HTTP Certificate resolver do not have any ingress reference or invalid ingress reference").Err()
	}
	return nil
}
