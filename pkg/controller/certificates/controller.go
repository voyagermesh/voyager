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
	aci "github.com/appscode/k8s-addons/api"
	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/k8s-addons/pkg/events"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/certificates/providers"
	"github.com/xenolf/lego/acme"
	"k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/util/intstr"
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

type CertificateController struct {
	// kubernetes clients
	KubeClient        clientset.Interface
	ACExtensionClient acs.AppsCodeExtensionInterface

	certificate       *aci.Certificate
	acmeCert          ACMECertData
	parsedCertificate *x509.Certificate
	sync.Mutex

	acmeClientConfig *ACMEConfig
	acmeClient       *ACMEClient

	userSecretName string
}

func NewController(c clientset.Interface, a acs.AppsCodeExtensionInterface) *CertificateController {
	return &CertificateController{
		KubeClient:        c,
		ACExtensionClient: a,
	}
}

func (c *CertificateController) Handle(e *events.Event) error {
	if e.ResourceType == events.Certificate {
		return c.handleCertificateEvent(e)
	}
	return c.handleIngressEvent(e)
}

func (c *CertificateController) handleCertificateEvent(e *events.Event) error {
	if e.EventType == events.Added || e.EventType == events.Updated {
		var cert *aci.Certificate

		// Indicates event contains an certificate to operate with
		ok := false
		switch e.EventType {
		case events.Added:
			cert, ok = e.RuntimeObj[0].(*aci.Certificate)
		case events.Updated:
			if len(e.RuntimeObj) > 1 {
				cert, ok = e.RuntimeObj[1].(*aci.Certificate)
			}
		}
		if ok {
			c.certificate = cert
			c.process(c.certificate)
		}
	}
	return nil
}

func (c *CertificateController) process(cert *aci.Certificate) error {
	c.acmeClientConfig = &ACMEConfig{
		Provider:      cert.Spec.Provider,
		ACMEServerUrl: cert.Spec.ACMEServerURL,
	}

	// Check if a cert already exists for this Certificate Instance
	secret, err := c.KubeClient.Core().Secrets(cert.Namespace).Get(defaultCertPrefix + cert.Name)
	if err == nil {
		var err error
		c.acmeCert, err = NewACMECertDataFromSecret(secret)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Error decoding acme certificate").Internal()
		}

		// Decode cert
		pemBlock, _ := pem.Decode(c.acmeCert.Cert)
		c.parsedCertificate, err = x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Error decoding x509 encoded certificate").Internal()
		}
		if !c.parsedCertificate.NotAfter.After(time.Now().Add(time.Hour * 24 * 7)) {
			log.Infoln("certificate is expiring in 7 days, attempting renew")
			c.renew()
		}

		if !c.acmeCert.EqualDomains(c.parsedCertificate) {
			c.renew()
		}
	}

	if k8serr.IsNotFound(err) || !cert.Status.CertificateObtained {
		// Certificate Not found as secret. We must create it now.
		c.create()
	}
	return nil
}

func (c *CertificateController) handleIngressEvent(e *events.Event) error {
	var ingress *aci.Ingress
	ok := false
	switch e.EventType {
	case events.Added:
		ingress, ok = e.RuntimeObj[0].(*aci.Ingress)
	case events.Updated:
		if len(e.RuntimeObj) > 1 {
			ingress, ok = e.RuntimeObj[1].(*aci.Ingress)
		}
	}

	if ok {
		if ingress.Annotations[certificateAnnotationKeyEnabled] == "true" {
			certificateName := ingress.Annotations[certificateAnnotationKeyName]
			// Check if a certificate already exists.
			certificate, err := c.ACExtensionClient.Certificate(ingress.Namespace).Get(certificateName)
			if err == nil {
				// Certificate exists mount it.
				return nil
			}
			if k8serr.IsNotFound(err) || !certificate.Status.CertificateObtained {
				newCertificate := &aci.Certificate{
					ObjectMeta: api.ObjectMeta{
						Name:      certificateName,
						Namespace: ingress.Namespace,
					},
					Spec: aci.CertificateSpec{
						Provider: ingress.Annotations[certificateAnnotationKeyProvider],
						Email:    ingress.Annotations[certificateAnnotationKeyEmail],
						ProviderCredentialSecretName: ingress.Annotations[certificateAnnotationKeyProviderCredentialSecretName],
						HTTPProviderIngressReference: api.ObjectReference{
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
				if v, ok := ingress.Annotations[aci.EngressKind]; ok {
					if v == "ingress" {
						newCertificate.Spec.HTTPProviderIngressReference.APIVersion = "extensions/v1beta1"
					} else {
						newCertificate.Spec.HTTPProviderIngressReference.APIVersion = "appscode.com/v1beta1"
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
				_, err := c.ACExtensionClient.Certificate(newCertificate.Namespace).Create(newCertificate)
				if err != nil {
					errors.New().WithCause(err).Internal()
				}
			}
		}
	}
	return nil
}

func (c *CertificateController) create() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.New().WithCause(err).Internal()
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
		log.Errorf("Error occured for %s, reason %s", k, v.Error())
	}
	if len(cert.PrivateKey) > 0 {
		return c.save(cert)
	}
	return nil
}

func (c *CertificateController) renew() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.New().WithCause(err).Internal()
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
		return errors.New().WithCause(err).Internal()
	}
	return c.update(cert)
}

func (c *CertificateController) ensureACMEClient() error {
	var acmeUserInfo *ACMEUserData
	acmeUserRegistered := false
	log.Infoln("trying to retrive acmeUser data")

	var userSecret *api.Secret
	err := errors.New().WithMessage("Setting error Not found").NotFound()
	if c.certificate.Spec.ACMEUserSecretName != "" {
		// ACMEUser secret name is provided.
		userSecret, err = c.KubeClient.Core().Secrets(c.certificate.Namespace).Get(c.certificate.Spec.ACMEUserSecretName)
	}
	if err != nil && c.certificate.Status.ACMEUserSecretName != "" {
		// There is a error getting the secret, try the secret name from status, if this was a update request
		userSecret, err = c.KubeClient.Core().Secrets(c.certificate.Namespace).Get(c.certificate.Status.ACMEUserSecretName)
	}
	if err != nil {
		// Trying to find an secret with the same name of Certificates
		userSecret, err = c.KubeClient.Core().Secrets(c.certificate.Namespace).Get(defaultUserSecretPrefix + c.certificate.Name)
		if err == nil {
			if _, ok := userSecret.Annotations[certificateKey+"/user-info"]; !ok {
				err = errors.New().WithMessagef("No %s annotaion key", certificateKey+"/user-info").Internal()
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
			return errors.New().WithCause(err).WithMessage("Failed to generate Key for New Acme User")
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
		return errors.New().WithCause(err).Internal()
	}

	log.V(9).Infoln("Getting NewACMECLient with config", c.acmeClientConfig)
	acmeClient, err := NewACMEClient(c.acmeClientConfig)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Failed to create acme client").Internal()
	}
	c.acmeClient = acmeClient

	if !acmeUserRegistered {
		return c.registerACMEUser(acmeClient)
	}
	return nil
}

func (c *CertificateController) registerACMEUser(acmeClient *ACMEClient) error {
	log.Debugln("ACME user not registered, registering new ACME user")
	registration, err := acmeClient.Register()
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Failed to registering user for new domain").Internal()
	}
	c.acmeClientConfig.UserData.Registration = registration
	if err := acmeClient.AgreeToTOS(); err != nil {
		return errors.New().WithCause(err).WithMessage("Failed to registering user for new domain").Internal()
	}

	// Acme User registered Create The acmeUserSecret
	secret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
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
	_, err = c.KubeClient.Core().Secrets(c.certificate.Namespace).Create(secret)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (c *CertificateController) loadProviderCredential() error {
	cred, err := c.KubeClient.Core().Secrets(c.certificate.Namespace).Get(c.certificate.Spec.ProviderCredentialSecretName)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	c.acmeClientConfig.ProviderCredentials = cred.Data
	return nil
}

func (c *CertificateController) save(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.certificate.Name, c.certificate.Namespace)
	_, err := c.KubeClient.Core().Secrets(c.certificate.Namespace).Create(secret)
	if err != nil {
		errors.New().WithCause(err).Internal()
	}

	k8sCert, err := c.ACExtensionClient.Certificate(c.certificate.Namespace).Get(c.certificate.Name)
	if err != nil {
		log.Errorln("failed to load cert object,", err)
	}

	// Update certificate data to add Details Information
	t := unversioned.Now()
	k8sCert.Status = aci.CertificateStatus{
		CertificateObtained: true,
		CreationTime:        &t,
		ACMEUserSecretName:  c.userSecretName,
		Details: aci.ACMECertificateDetails{
			Domain:        cert.Domain,
			CertURL:       cert.CertURL,
			CertStableURL: cert.CertStableURL,
			AccountRef:    cert.AccountRef,
		},
	}
	c.ACExtensionClient.Certificate(c.certificate.Namespace).Update(k8sCert)
	return nil
}

func (c *CertificateController) update(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.certificate.Name, c.certificate.Namespace)
	oldSecret, err := c.KubeClient.Core().Secrets(c.certificate.Namespace).Get(secret.Name)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	oldSecret.Data = secret.Data
	_, err = c.KubeClient.Core().Secrets(c.certificate.Namespace).Update(oldSecret)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (c *CertificateController) processHTTPCertificate(revert chan struct{}) error {
	c.acmeClient.HTTPProviderLock.Lock()
	defer c.acmeClient.HTTPProviderLock.Unlock()

	switch c.certificate.Spec.HTTPProviderIngressReference.APIVersion {
	case "appscode.com/v1":
		revertRequired := false
		i, err := c.ACExtensionClient.Ingress(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
			Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		// make a copy of previous spec.
		prevSpecs := i.Spec
		for _, host := range c.certificate.Spec.Domains {
			rule := aci.ExtendedIngressRule{
				Host: host,
				ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
					HTTP: &aci.HTTPExtendedIngressRuleValue{
						Paths: []aci.HTTPExtendedIngressPath{
							{
								Path: providers.URLPrefix,
								Backend: aci.ExtendedIngressBackend{
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
		_, err = c.ACExtensionClient.Ingress(c.certificate.Namespace).Update(i)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.ACExtensionClient.Ingress(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
						Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, aci.ExtendedIngressTLS{
							Hosts:      c.certificate.Spec.Domains,
							SecretName: defaultCertPrefix + c.certificate.Name,
						})
						c.ACExtensionClient.Ingress(c.certificate.Namespace).Update(i)
					}
				}
				return
			}
		}()
	case "extensions/v1beta1":
		revertRequired := false
		i, err := c.KubeClient.Extensions().Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
			Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
		if err != nil {
			return errors.New().WithCause(err).Internal()
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
		_, err = c.KubeClient.Extensions().Ingresses(c.certificate.Namespace).Update(i)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.KubeClient.Extensions().Ingresses(c.certificate.Spec.HTTPProviderIngressReference.Namespace).
						Get(c.certificate.Spec.HTTPProviderIngressReference.Name)
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, extensions.IngressTLS{
							Hosts:      c.certificate.Spec.Domains,
							SecretName: defaultCertPrefix + c.certificate.Name,
						})
						c.KubeClient.Extensions().Ingresses(c.certificate.Namespace).Update(i)
					}
				}
				return
			}
		}()
	default:
		return errors.New().WithMessage("HTTP Certificate resolver do not have any ingress refrence or invalid ingress refrences").NotFound()
	}
	return nil
}
