package certificate

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
	tapi "github.com/appscode/voyager/api"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/appscode/voyager/pkg/util"
	"github.com/xenolf/lego/acme"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

const (
	defaultCertPrefix       = "cert-"
	defaultUserSecretPrefix = "acme-"
)

type Controller struct {
	KubeConfig *rest.Config
	KubeClient clientset.Interface
	ExtClient  acs.ExtensionInterface
	Opt        config.Options
	recorder   record.EventRecorder

	tpr                        *api.Certificate
	acmeCert                   ACMECertData
	crt                        *x509.Certificate
	renewedCertificateResource acme.CertificateResource
	sync.Mutex

	acmeClientConfig *ACMEConfig
	acmeClient       *ACMEClient

	userSecretName string
}

func NewController(config *rest.Config, kubeClient clientset.Interface, extClient acs.ExtensionInterface, opt config.Options, tpr *api.Certificate) *Controller {
	return &Controller{
		KubeConfig: config,
		KubeClient: kubeClient,
		ExtClient:  extClient,
		Opt:        opt,
		tpr:        tpr,
		recorder:   eventer.NewEventRecorder(kubeClient, "Voyager operator"),
	}
}

func (c *Controller) HandleIngress(ingress *api.Ingress) error {
	if ingress.Annotations != nil {
		if cert, ok := ingress.CertificateSpec(); ok {
			issuedCert, err := c.ExtClient.Certificates(ingress.Namespace).Get(cert.Name)
			if err == nil {
				// Certificate exists mount it.
				return nil
			}

			if kerr.IsNotFound(err) || !issuedCert.Status.CertificateObtained {
				_, err := c.ExtClient.Certificates(cert.Namespace).Create(cert)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Controller) Process() error {
	c.acmeClientConfig = &ACMEConfig{
		Provider:            c.tpr.Spec.Provider,
		ACMEServerUrl:       c.tpr.Spec.ACMEServerURL,
		ProviderCredentials: make(map[string][]byte),
	}

	c.acmeCert.Domains = NewDomainCollection(c.tpr.Spec.Domains...)
	// Check if a cert already exists for this Certificate Instance
	secret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(defaultCertPrefix+c.tpr.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) || !c.tpr.Status.CertificateObtained {
		// Certificate Not found as secret. We must create it now.
		err := c.create()
		if err != nil {
			c.recorder.Eventf(
				c.tpr,
				apiv1.EventTypeWarning,
				eventer.EventReasonCertificateCreateFailed,
				"Failed to create certificate, Reason: %s",
				err.Error(),
			)
			return err
		}
		c.recorder.Eventf(
			c.tpr,
			apiv1.EventTypeNormal,
			eventer.EventReasonCertificateCreateSuccessful,
			"Successfully created certificate",
		)
		return nil
	}

	// Secret is exists, try to renew certificate if necessary
	c.acmeCert, err = NewACMECertDataFromSecret(secret, c.tpr)
	if err != nil {
		return errors.FromErr(err).WithMessage("Error decoding acme certificate").Err()
	}

	// Decode cert
	pemBlock, _ := pem.Decode(c.acmeCert.Cert)
	c.crt, err = x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errors.FromErr(err).WithMessage("Error decoding x509 encoded certificate").Err()
	}

	c.acmeCert.Domains = NewDomainCollection(c.tpr.Spec.Domains...)
	if !c.crt.NotAfter.After(time.Now().Add(time.Hour*24*7)) ||
		!c.acmeCert.EqualDomains(c.crt) {
		log.Infof(
			"Trying to renew certificate for, tpr domains %v, cert dns %v, cert common name %v",
			c.tpr.Spec.Domains,
			c.crt.DNSNames,
			c.crt.Subject.CommonName,
		)
		err := c.renew()
		if err != nil {
			c.recorder.Eventf(
				c.tpr,
				apiv1.EventTypeWarning,
				eventer.EventReasonCertificateRenewFailed,
				"Failed to renew certificate, Reason %s",
				err.Error(),
			)
			return err
		}
		c.recorder.Eventf(
			c.tpr,
			apiv1.EventTypeNormal,
			eventer.EventReasonCertificateRenewSuccessful,
			"Successfully renewed certificate, voyager pods that mount this secret needs to restart.",
		)

		// Try to restart every HAProxy that mount this cert
		go c.restartHAProxyIfRequired()
	}
	return nil
}

func (c *Controller) create() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.FromErr(err).Err()
	}
	if c.tpr.Spec.Provider == "http" {
		done := make(chan struct{})
		defer func() {
			done <- struct{}{}
			close(done)
		}()
		if err := c.processHTTPCertificate(done); err != nil {
			return err
		}
	}
	cert, errs := c.acmeClient.ObtainCertificate(c.tpr.Spec.Domains, true, nil, false)
	for k, v := range errs {
		log.Errorf("Error occurred for %s, reason %s", k, v.Error())
	}
	if len(cert.PrivateKey) > 0 {
		return c.save(cert)
	}
	return nil
}

func (c *Controller) renew() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.FromErr(err).Err()
	}

	if c.tpr.Spec.Provider == "http" {
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
		Domain:        c.tpr.Status.Details.Domain,
		CertURL:       c.tpr.Status.Details.CertURL,
		CertStableURL: c.tpr.Status.Details.CertStableURL,
		AccountRef:    c.tpr.Status.Details.AccountRef,
		Certificate:   c.acmeCert.Cert,
		PrivateKey:    c.acmeCert.PrivateKey,
	}
	cert, err := c.acmeClient.RenewCertificate(acmeCert, true, false)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.renewedCertificateResource = cert
	return c.update(cert)
}

func (c *Controller) ensureACMEClient() error {
	var acmeUserInfo *ACMEUserData
	acmeUserRegistered := false
	log.Infoln("trying to retrive acmeUser data")

	var userSecret *apiv1.Secret
	err := errors.New("Setting error Not found").Err()
	if c.tpr.Spec.ACMEUserSecretName != "" {
		// ACMEUser secret name is provided.
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(c.tpr.Spec.ACMEUserSecretName, metav1.GetOptions{})
	}
	if err != nil && c.tpr.Status.ACMEUserSecretName != "" {
		// There is a error getting the secret, try the secret name from status, if this was a update request
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(c.tpr.Status.ACMEUserSecretName, metav1.GetOptions{})
	}
	if err != nil {
		// Trying to find an secret with the same name of Certificates
		userSecret, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(defaultUserSecretPrefix+c.tpr.Name, metav1.GetOptions{})
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
			Email: c.tpr.Spec.Email,
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

func (c *Controller) registerACMEUser(acmeClient *ACMEClient) error {
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
			Name:      c.tpr.Spec.ACMEUserSecretName,
			Namespace: c.tpr.Namespace,
			Labels: map[string]string{
				certificateKey + "/user-info": "true",
				certificateKey + "/cert-name": c.tpr.Name,
			},
			Annotations: map[string]string{
				certificateKey + "/user-info": "true",
				certificateKey + "/cert-name": c.tpr.Name,
			},
		},
		Data: map[string][]byte{
			"user-info": c.acmeClientConfig.UserData.Json(),
		},
		Type: "certificate.appscode.com/acme-user-info",
	}
	if secret.Name == "" {
		secret.Name = defaultUserSecretPrefix + c.tpr.Name
	}
	c.userSecretName = secret.Name
	log.Debugln("User Registered saving User Informations with Secret name", c.userSecretName)
	_, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Create(secret)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Controller) loadProviderCredential() error {
	if len(c.tpr.Spec.ProviderCredentialSecretName) > 0 {
		cred, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(c.tpr.Spec.ProviderCredentialSecretName, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}
		c.acmeClientConfig.ProviderCredentials = cred.Data
	}
	return nil
}

func (c *Controller) save(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.tpr.Name, c.tpr.Namespace)
	_, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Create(secret)
	if err != nil {
		errors.FromErr(err).Err()
	}

	k8sCert, err := c.ExtClient.Certificates(c.tpr.Namespace).Get(c.tpr.Name)
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
	c.ExtClient.Certificates(c.tpr.Namespace).Update(k8sCert)
	return nil
}

func (c *Controller) update(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	secret := certData.ToSecret(c.tpr.Name, c.tpr.Namespace)
	oldSecret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	oldSecret.Data = secret.Data
	_, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Update(oldSecret)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Controller) processHTTPCertificate(revert chan struct{}) error {
	c.acmeClient.HTTPProviderLock.Lock()
	defer c.acmeClient.HTTPProviderLock.Unlock()

	switch c.tpr.Spec.HTTPProviderIngressReference.APIVersion {
	case api.V1beta1SchemeGroupVersion.String():
		revertRequired := false
		i, err := c.ExtClient.Ingresses(c.tpr.Spec.HTTPProviderIngressReference.Namespace).
			Get(c.tpr.Spec.HTTPProviderIngressReference.Name)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		// make a copy of previous spec.
		prevSpecs := i.Spec
		for _, host := range c.tpr.Spec.Domains {
			rule := api.IngressRule{
				Host: host,
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
		}
		_, err = c.ExtClient.Ingresses(c.tpr.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.ExtClient.Ingresses(c.tpr.Spec.HTTPProviderIngressReference.Namespace).
						Get(c.tpr.Spec.HTTPProviderIngressReference.Name)
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, api.IngressTLS{
							Hosts:      c.tpr.Spec.Domains,
							SecretName: defaultCertPrefix + c.tpr.Name,
						})
						c.ExtClient.Ingresses(c.tpr.Namespace).Update(i)
					}
				}
				return
			}
		}()
	case "extensions/v1beta1":
		revertRequired := false
		i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Spec.HTTPProviderIngressReference.Namespace).
			Get(c.tpr.Spec.HTTPProviderIngressReference.Name, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}
		// make a copy of previous spec.
		prevSpecs := i.Spec
		for _, host := range c.tpr.Spec.Domains {
			rule := extensions.IngressRule{
				Host: host,
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
		}
		_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		revertRequired = true
		// After All is done revert everything
		defer func() {
			select {
			case <-revert:
				if revertRequired {
					i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Spec.HTTPProviderIngressReference.Namespace).
						Get(c.tpr.Spec.HTTPProviderIngressReference.Name, metav1.GetOptions{})
					if err == nil {
						i.Spec = prevSpecs
						i.Spec.TLS = append(i.Spec.TLS, extensions.IngressTLS{
							Hosts:      c.tpr.Spec.Domains,
							SecretName: defaultCertPrefix + c.tpr.Name,
						})
						c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Namespace).Update(i)
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

func (c *Controller) restartHAProxyIfRequired() {
	renewedCert, _ := pem.Decode(c.renewedCertificateResource.Certificate)
	parsedRenewedCert, err := x509.ParseCertificate(renewedCert.Bytes)
	if err != nil {
		log.Errorln("Failed starting HAProxy reload", err)
		return
	}

	ing, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorln(err)
		return
	}
	eng, err := c.ExtClient.Ingresses(c.tpr.Namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorln(err)
		return
	}

	items := make([]tapi.Ingress, len(ing.Items))
	for i, item := range ing.Items {
		e, err := tapi.NewEngressFromIngress(item)
		if err != nil {
			log.Errorln(err)
			continue
		}
		items[i] = *e
	}
	items = append(items, eng.Items...)
	for _, ing := range items {
		if strings.Contains(ing.Secrets(), defaultCertPrefix+c.tpr.Name) {
			podList, err := c.KubeClient.CoreV1().Pods(ing.Namespace).List(metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(labels.Set(ing.OffshootLabels())).String(),
			})
			if err == nil {
				for _, pod := range podList.Items {
					for range time.NewTicker(time.Second * 20).C {
						out := util.Exec(
							c.KubeClient.CoreV1().RESTClient(),
							c.KubeConfig,
							pod,
							[]string{"cat /srv/haproxy/secrets/" + defaultCertPrefix + c.tpr.Name + "/tls.crt"},
						)

						pemBlock, _ := pem.Decode([]byte(out))
						parsedCert, err := x509.ParseCertificate(pemBlock.Bytes)
						if err != nil {
							log.Errorln(err)
							continue
						}

						if parsedCert.Equal(parsedRenewedCert) {
							util.Exec(
								c.KubeClient.CoreV1().RESTClient(),
								c.KubeConfig,
								pod,
								[]string{"/etc/sv/reloader/restart"},
							)
							break
						}
					}
				}
			}
		}
	}
}
