package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	"github.com/appscode/go/strings"
	tapi "github.com/appscode/voyager/apis/voyager/v1beta1"
	tapi_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
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
	defaultCertPrefix = "cert-"
)

type Controller struct {
	KubeConfig *rest.Config
	KubeClient clientset.Interface
	ExtClient  acs.VoyagerV1beta1Interface
	Opt        config.Options
	recorder   record.EventRecorder

	tpr                        *tapi.Certificate
	acmeCert                   ACMECertData
	crt                        *x509.Certificate
	renewedCertificateResource acme.CertificateResource
	sync.Mutex

	acmeConfig *ACMEConfig
	acmeClient *ACMEClient
}

func NewController(config *rest.Config, kubeClient clientset.Interface, extClient acs.VoyagerV1beta1Interface, opt config.Options, tpr *tapi.Certificate) *Controller {
	return &Controller{
		KubeConfig: config,
		KubeClient: kubeClient,
		ExtClient:  extClient,
		Opt:        opt,
		tpr:        tpr,
		recorder:   eventer.NewEventRecorder(kubeClient, "Voyager operator"),
	}
}

func (c *Controller) Process() error {
	cfg := &ACMEConfig{
		CloudProvider:  c.Opt.CloudProvider,
		DNSCredentials: make(map[string][]byte),
		UserSecret:     ACMEUserSecret{},
	}
	if c.tpr.Spec.ChallengeProvider.HTTP != nil {
		cfg.ChallengeProvider = "http"
	} else if c.tpr.Spec.ChallengeProvider.DNS != nil {
		cfg.ChallengeProvider = c.tpr.Spec.ChallengeProvider.DNS.ProviderType
		if len(c.tpr.Spec.ChallengeProvider.DNS.CredentialSecretName) > 0 {
			cred, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(
				c.tpr.Spec.ChallengeProvider.DNS.CredentialSecretName,
				metav1.GetOptions{},
			)
			if err != nil {
				c.recorder.Eventf(
					c.tpr.ObjectReference(),
					apiv1.EventTypeWarning,
					eventer.EventReasonCertificateCreateFailed,
					"Failed to create certificate, Reason: %s",
					err.Error(),
				)
				return err
			}
			cfg.DNSCredentials = cred.Data
		}
	}
	c.acmeConfig = cfg

	c.acmeCert.Domains = NewDomainCollection(c.tpr.Spec.Domains...)
	// Check if a cert already exists for this Certificate Instance
	secret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(defaultCertPrefix+c.tpr.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) || !c.tpr.Status.CertificateObtained {
		// Certificate Not found as secret. We must create it now.
		err := c.create()
		if err != nil {
			c.recorder.Eventf(
				c.tpr.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonCertificateCreateFailed,
				"Failed to create certificate, Reason: %s",
				err.Error(),
			)
			return err
		}
		c.recorder.Eventf(
			c.tpr.ObjectReference(),
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
				c.tpr.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonCertificateRenewFailed,
				"Failed to renew certificate, Reason %s",
				err.Error(),
			)
			return err
		}
		c.recorder.Eventf(
			c.tpr.ObjectReference(),
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
		return err
	}
	if c.acmeConfig.ChallengeProvider == "http" {
		if err := c.processHTTPCertificate(); err != nil {
			return err
		}
	}
	cert, errs := c.acmeClient.ObtainCertificate(c.tpr.Spec.Domains, true, nil, true)
	for k, v := range errs {
		log.Errorf("Error occurred for %s, reason %s", k, v.Error())
	}
	if len(cert.PrivateKey) > 0 {
		return c.save(cert)
	}
	return errors.New("failed to create certificate")
}

func (c *Controller) renew() error {
	if err := c.ensureACMEClient(); err != nil {
		return errors.FromErr(err).Err()
	}

	if c.acmeConfig.ChallengeProvider == "http" {
		if err := c.processHTTPCertificate(); err != nil {
			return err
		}
	}
	acmeCert := acme.CertificateResource{
		CertURL:       c.tpr.Status.Certificate.CertURL,
		CertStableURL: c.tpr.Status.Certificate.CertStableURL,
		AccountRef:    c.tpr.Status.Certificate.AccountRef,
		Certificate:   c.acmeCert.Cert,
		PrivateKey:    c.acmeCert.PrivateKey,
	}
	cert, err := c.acmeClient.RenewCertificate(acmeCert, true, true)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.renewedCertificateResource = cert
	return c.update(cert)
}

func (c *Controller) ensureACMEClient() error {
	secret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(c.tpr.Spec.ACMEUserSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if _, found := secret.Data[tapi.ACMEUserEmail]; !found {
		return err
	}
	c.acmeConfig.UserSecret = ACMEUserSecret(secret.Data)

	// No error that means we successfully got an userSecret
	var acmeUserInfo *ACMEUserData
	acmeUserRegistered := false
	if data := c.acmeConfig.UserSecret.GetUserData(); len(data) > 0 {
		acmeUserInfo = &ACMEUserData{}
		userError := json.Unmarshal(data, acmeUserInfo)
		if userError == nil {
			acmeUserRegistered = true
		}
	}

	if !acmeUserRegistered {
		log.Infoln("No ACME user found, Generate a new ACME user")
		userKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errors.FromErr(err).WithMessage("Failed to generate Key for New Acme User")
		}
		acmeUserInfo = &ACMEUserData{
			Email: c.acmeConfig.UserSecret.GetEmail(),
			Key: pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(userKey),
			}),
		}
	}
	c.acmeConfig.UserData = acmeUserInfo
	acmeClient, err := NewACMEClient(c.acmeConfig)
	if err != nil {
		return fmt.Errorf("failed to create acme client. Reason: %s", err)
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
	c.acmeConfig.UserData.Registration = registration
	if err := acmeClient.AgreeToTOS(); err != nil {
		return errors.FromErr(err).WithMessage("Failed to registering user for new domain").Err()
	}

	secret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(c.tpr.Spec.ACMEUserSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	secret.Data[tapi.ACMEUserDataJSON] = c.acmeConfig.UserData.Json()
	c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Update(secret)
	return nil
}

func (c *Controller) save(cert acme.CertificateResource) error {
	certData := &ACMECertData{
		Domains:    c.acmeCert.Domains,
		Cert:       cert.Certificate,
		PrivateKey: cert.PrivateKey,
	}

	if c.tpr.Spec.Storage.Kubernetes != nil {
		secret := certData.ToSecret(c.tpr.Name, c.tpr.Namespace, c.tpr.Spec.Storage.Kubernetes.Name)
		_, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Create(secret)
		if err != nil {
			errors.FromErr(err).Err()
		}
	}

	k8sCert, err := c.ExtClient.Certificates(c.tpr.Namespace).Get(c.tpr.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorln("failed to load cert object,", err)
	}

	// Decode cert
	pemBlock, _ := pem.Decode(cert.Certificate)
	crt, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errors.FromErr(err).WithMessage("Error decoding x509 encoded certificate").Err()
	}

	// Update certificate data to add Details Information
	t := metav1.Now()
	k8sCert.Status = tapi.CertificateStatus{
		CertificateObtained: true,
		CreationTime:        &t,
		Conditions: []tapi.CertificateCondition{{
			Type:           tapi.CertificateCreated,
			LastUpdateTime: t,
		}},
		Certificate: tapi.ACMECertificateDetails{
			SerialNumber:  crt.SerialNumber.String(),
			NotBefore:     metav1.NewTime(crt.NotBefore),
			NotAfter:      metav1.NewTime(crt.NotAfter),
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

	if c.tpr.Spec.Storage.Kubernetes != nil {
		secret := certData.ToSecret(c.tpr.Name, c.tpr.Namespace, c.tpr.Spec.Storage.Kubernetes.Name)
		oldSecret, err := c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Get(secret.Name, metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}
		oldSecret.Data = secret.Data
		_, err = c.KubeClient.CoreV1().Secrets(c.tpr.Namespace).Update(oldSecret)
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}

	k8sCert, err := c.ExtClient.Certificates(c.tpr.Namespace).Get(c.tpr.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorln("failed to load cert object,", err)
	}

	// Decode cert
	pemBlock, _ := pem.Decode(cert.Certificate)
	crt, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return errors.FromErr(err).WithMessage("Error decoding x509 encoded certificate").Err()
	}

	// Update certificate data to add Details Information
	t := metav1.Now()
	k8sCert.Status.Certificate = tapi.ACMECertificateDetails{
		SerialNumber:  crt.SerialNumber.String(),
		NotBefore:     metav1.NewTime(crt.NotBefore),
		NotAfter:      metav1.NewTime(crt.NotAfter),
		CertURL:       cert.CertURL,
		CertStableURL: cert.CertStableURL,
		AccountRef:    cert.AccountRef,
	}

	found := false
	for i := range k8sCert.Status.Conditions {
		if k8sCert.Status.Conditions[i].Type == tapi.CertificateUpdated {
			k8sCert.Status.Conditions[i].LastUpdateTime = t
			found = true
		}
	}
	if !found {
		k8sCert.Status.Conditions = append(k8sCert.Status.Conditions, tapi.CertificateCondition{
			Type:           tapi.CertificateUpdated,
			LastUpdateTime: t,
		})
	}

	c.ExtClient.Certificates(c.tpr.Namespace).Update(k8sCert)
	return nil
}

func (c *Controller) processHTTPCertificate() error {
	switch c.tpr.Spec.ChallengeProvider.HTTP.Ingress.APIVersion {
	case tapi_v1beta1.SchemeGroupVersion.String():
		i, err := c.ExtClient.Ingresses(c.tpr.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
			Get(c.tpr.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
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

		rule := tapi.IngressRule{
			IngressRuleValue: tapi.IngressRuleValue{
				HTTP: &tapi.HTTPIngressRuleValue{
					Paths: []tapi.HTTPIngressPath{
						{
							Path: providers.URLPrefix,
							Backend: tapi.HTTPIngressBackend{
								IngressBackend: tapi.IngressBackend{
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

		_, err = c.ExtClient.Ingresses(c.tpr.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
	case "extensions/v1beta1":
		i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Spec.ChallengeProvider.HTTP.Ingress.Namespace).
			Get(c.tpr.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
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

		_, err = c.KubeClient.ExtensionsV1beta1().Ingresses(c.tpr.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
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
