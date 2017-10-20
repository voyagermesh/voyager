package certificate

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	v1u "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	vu "github.com/appscode/voyager/client/typed/voyager/v1beta1/util"
	"github.com/appscode/voyager/pkg/certificate/providers"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/eventer"
	"github.com/xenolf/lego/acme"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/cert"
)

type Controller struct {
	KubeClient    clientset.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	Opt           config.Options
	recorder      record.EventRecorder

	crd               *api.Certificate
	ChallengeProvider string
	UserSecret        *apiv1.Secret
	DNSCredentials    map[string][]byte
	curCert           *x509.Certificate
	acmeUser          *ACMEUser
	acmeClient        *acme.Client
	store             *CertStore
	logger            *log.Logger
}

func NewController(ctx context.Context, kubeClient clientset.Interface, extClient acs.VoyagerV1beta1Interface, opt config.Options, tpr *api.Certificate) (*Controller, error) {
	ctrl := &Controller{
		logger:        log.New(ctx),
		KubeClient:    kubeClient,
		VoyagerClient: extClient,
		Opt:           opt,
		crd:           tpr,
		recorder:      eventer.NewEventRecorder(kubeClient, "voyager-operator"),
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
			_, err = ctrl.VoyagerClient.Ingresses(ctrl.crd.Namespace).
				Get(ctrl.crd.Spec.ChallengeProvider.HTTP.Ingress.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
		case "extensions/v1beta1":
			ing, err := ctrl.KubeClient.ExtensionsV1beta1().Ingresses(ctrl.crd.Namespace).
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
			ctrl.DNSCredentials = dnsSecret.Data
		}
	}

	ctrl.store, err = NewCertStore(kubeClient, extClient)
	if err != nil {
		return nil, err
	}
	if ctrl.store.VaultClient == nil && ctrl.crd.Spec.Storage.Vault != nil {
		return nil, fmt.Errorf("certificate %s@%s uses vault but vault address is missing", tpr.Name, tpr.Namespace)
	}

	return ctrl, nil
}

func (c *Controller) Process() error {
	pemCrt, _, err := c.store.Get(c.crd)
	if err != nil {
		return err
	}
	if pemCrt == nil {
		err := c.create()
		if err == nil {
			c.recorder.Eventf(
				c.crd.ObjectReference(),
				apiv1.EventTypeNormal,
				eventer.EventReasonCertificateIssueSuccessful,
				"Successfully issued certificate",
			)
		}
		return err
	}

	var certs []*x509.Certificate
	certs, err = cert.ParseCertsPEM(pemCrt)
	if err != nil {
		return fmt.Errorf("secret %s@%s contains bad certificate. Reason: %s", c.crd.SecretName(), c.crd.Namespace, err)
	}
	c.curCert = certs[0]
	if c.crd.ShouldRenew(c.curCert) {
		err := c.renew()
		if err == nil {
			c.recorder.Eventf(
				c.crd.ObjectReference(),
				apiv1.EventTypeNormal,
				eventer.EventReasonCertificateIssueSuccessful,
				"Successfully renewed certificate",
			)
		}
		return err
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
		c.logger.Infoln("No ACME user found, registering a new ACME user")
		userKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate key for Acme User")
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
		c.acmeUser.Registration = registration
		if err := c.acmeClient.AgreeToTOS(); err != nil {
			return fmt.Errorf("failed to register user %s. Reason: %s", c.acmeUser.Email, err)
		}
		c.UserSecret, err = v1u.PatchSecret(c.KubeClient, c.UserSecret, func(in *apiv1.Secret) *apiv1.Secret {
			if in.Data == nil {
				in.Data = make(map[string][]byte)
			}
			in.Data[api.ACMEUserPrivatekey] = cert.EncodePrivateKeyPEM(c.acmeUser.Key.(*rsa.PrivateKey))
			regBytes, _ := json.Marshal(c.acmeUser.Registration)
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
		if err := c.updateIngress(); err != nil {
			return err
		}
	}
	cert, errs := c.acmeClient.ObtainCertificate(c.crd.Spec.Domains, true, nil, false)
	if len(errs) > 0 {
		causes := make([]string, 0, len(errs))
		for k, v := range errs {
			causes = append(causes, k+": "+v.Error())
		}
		return c.processError(fmt.Errorf("failed to create certificate. Reason: %s", strings.Join(causes, ", ")))
	}
	return c.store.Save(c.crd, cert)
}

func (c *Controller) renew() error {
	if err := c.getACMEClient(); err != nil {
		return err
	}
	if c.ChallengeProvider == "http" {
		if err := c.updateIngress(); err != nil {
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
	cert, err := c.acmeClient.RenewCertificate(acmeCert, true, false)
	if err != nil {
		return c.processError(err)
	}
	return c.store.Save(c.crd, cert)
}

func (c *Controller) processError(err error) error {
	vu.PatchCertificate(c.VoyagerClient, c.crd, func(in *api.Certificate) *api.Certificate {
		// Update certificate data to add Details Information
		t := metav1.Now()
		found := false
		condType := api.CertificateFailed
		if strings.Contains(err.Error(), "urn:acme:error:rateLimited") {
			condType = api.CertificateRateLimited
		}
		for i := range in.Status.Conditions {
			if in.Status.Conditions[i].Type == condType {
				in.Status.Conditions[i].LastUpdateTime = t
				in.Status.Conditions[i].Reason = err.Error()
				found = true
			}
		}
		if !found {
			in.Status.Conditions = append(in.Status.Conditions, api.CertificateCondition{
				Type:           condType,
				LastUpdateTime: t,
				Reason:         err.Error(),
			})
		}
		return in
	})
	return err
}

func (c *Controller) updateIngress() error {
	switch c.crd.Spec.ChallengeProvider.HTTP.Ingress.APIVersion {
	case api.SchemeGroupVersion.String():
		i, err := c.VoyagerClient.Ingresses(c.crd.Namespace).
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
									ServicePort: intstr.FromInt(providers.ACMEResponderPort),
								},
							},
						},
					},
				},
			},
		}
		i.Spec.Rules = append([]api.IngressRule{rule}, i.Spec.Rules...)

		_, err = c.VoyagerClient.Ingresses(c.crd.Namespace).Update(i)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		time.Sleep(time.Second * 5)
	case "extensions/v1beta1":
		i, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.crd.Namespace).
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
								ServicePort: intstr.FromInt(providers.ACMEResponderPort),
							},
						},
					},
				},
			},
		}
		i.Spec.Rules = append([]extensions.IngressRule{rule}, i.Spec.Rules...)

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
