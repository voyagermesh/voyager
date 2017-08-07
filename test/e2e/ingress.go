package e2e

/*
import (
	"net/http"
	"strings"
	"time"
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)


func (s *IngressTestSuit) TestIngressKeepSource() error {
	baseIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.KeepSourceIP: "true",
			},
		},
		Spec: api.IngressSpec{
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.IngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *apiv1.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = s.t.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	serverAddr, err := s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	return nil
}

func (s *IngressTestSuit) TestIngressLBSourceRange() error {
	baseIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.KeepSourceIP: "true",
			},
		},
		Spec: api.IngressSpec{
			LoadBalancerSourceRanges: []string{
				"192.101.0.0/16",
				"192.0.0.0/24",
			},
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.IngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *apiv1.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if len(svc.Spec.LoadBalancerSourceRanges) != 2 {
		return errors.New().WithMessage("LBSource range did not matched").Err()
	}

	tobeUpdated, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	tobeUpdated.Spec.LoadBalancerSourceRanges = []string{"192.10.0.0/24"}
	_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Update(tobeUpdated)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if len(svc.Spec.LoadBalancerSourceRanges) != 1 {
		return errors.New().WithMessage("LBSource range did not matched").Err()
	}
	return nil
}

func (s *IngressTestSuit) TestIngressExternalNameResolver() error {
	extSvcResolvesDNSWithNS := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-svc-dns-with-ns",
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.UseDNSResolver:         "true",
				api.DNSResolverNameservers: `["8.8.8.8:53", "8.8.4.4:53"]`,
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:         apiv1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err := s.t.KubeClient.CoreV1().Services(extSvcResolvesDNSWithNS.Namespace).Create(extSvcResolvesDNSWithNS)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.CoreV1().Services(extSvcResolvesDNSWithNS.Namespace).Delete(extSvcResolvesDNSWithNS.Name, nil)
		}
	}()

	extSvcNoResolveRedirect := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-svc-non-dns",
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: apiv1.ServiceSpec{
			Type:         apiv1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err = s.t.KubeClient.CoreV1().Services(extSvcNoResolveRedirect.Namespace).Create(extSvcNoResolveRedirect)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.CoreV1().Services(extSvcNoResolveRedirect.Namespace).Delete(extSvcNoResolveRedirect.Name, nil)
		}
	}()

	extSvcResolvesDNSWithoutNS := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-svc-dns",
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: apiv1.ServiceSpec{
			Type:         apiv1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err = s.t.KubeClient.CoreV1().Services(extSvcResolvesDNSWithoutNS.Namespace).Create(extSvcResolvesDNSWithoutNS)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.CoreV1().Services(extSvcResolvesDNSWithoutNS.Namespace).Delete(extSvcResolvesDNSWithoutNS.Name, nil)
		}
	}()

	baseIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.IngressSpec{
			Backend: &api.IngressBackend{
				ServiceName: extSvcNoResolveRedirect.Name,
				ServicePort: intstr.FromString("80"),
			},
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/test-dns",
									Backend: api.IngressBackend{
										ServiceName: extSvcResolvesDNSWithNS.Name,
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/test-no-dns",
									Backend: api.IngressBackend{
										ServiceName: extSvcNoResolveRedirect.Name,
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/test-no-backend-redirect",
									Backend: api.IngressBackend{
										ServiceName: extSvcResolvesDNSWithoutNS.Name,
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/test-no-backend-rule-redirect",
									Backend: api.IngressBackend{
										ServiceName: extSvcNoResolveRedirect.Name,
										ServicePort: intstr.FromString("80"),
										BackendRule: []string{
											"http-request redirect location https://google.com code 302",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *apiv1.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return err
	}
	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = s.t.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 60)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	// Check Non DNS redirect
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/test-no-dns").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
		if resp.ResponseHeader.Get("Location") != "http://google.com:80" {
			return errors.New().WithMessage("Location did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/test-no-backend-redirect").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
		if resp.ResponseHeader.Get("Location") != "http://google.com:80" {
			return errors.New().WithMessage("Location did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/test-no-backend-rule-redirect").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}

		if resp.Status != 302 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
		if resp.ResponseHeader.Get("Location") != "https://google.com" {
			return errors.New().WithMessage("Location did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/test-dns").DoStatusWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 404 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/default").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
		if resp.ResponseHeader.Get("Location") != "http://google.com:80" {
			return errors.New().WithMessage("Location did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressExternalNameWithBackendRules() error {
	extSvcNoResolveRedirect := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-svc-non-dns",
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: apiv1.ServiceSpec{
			Type:         apiv1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err := s.t.KubeClient.CoreV1().Services(extSvcNoResolveRedirect.Namespace).Create(extSvcNoResolveRedirect)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.CoreV1().Services(extSvcNoResolveRedirect.Namespace).Delete(extSvcNoResolveRedirect.Name, nil)
		}
	}()

	baseIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.IngressSpec{
			Backend: &api.IngressBackend{
				ServiceName: testServerSvc.Name,
				ServicePort: intstr.FromString("8989"),
			},
			Rules: []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/redirect-rule",
									Backend: api.IngressBackend{
										BackendRule: []string{
											"http-request redirect location https://github.com/appscode/discuss/issues code 301",
										},
										ServiceName: extSvcNoResolveRedirect.Name,
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/redirect",
									Backend: api.IngressBackend{
										ServiceName: extSvcNoResolveRedirect.Name,
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/back-end",
									Backend: api.IngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromString("8989"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *apiv1.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return err
	}
	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = s.t.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 60)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/redirect-rule").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
	}

	err = s.t.KubeClient.CoreV1().Pods(s.t.Config.TestNamespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				map[string]string{
					"app": "test-server",
				},
			).String(),
		},
	)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = s.t.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err = s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 60)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/redirect-rule").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Code did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressOperatorWithRBAC() error {
	if s.t.Config.RBACEnabled && s.t.Config.InCluster {
		baseIngress := &api.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testIngressName(),
				Namespace: s.t.Config.TestNamespace,
			},
			Spec: api.IngressSpec{
				Rules: []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.IngressBackend{
											ServiceName: testServerSvc.Name,
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		defer func() {
			if s.t.Config.Cleanup {
				s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		var svc *apiv1.Service
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for service to be created")
		}
		if err != nil {
			return err
		}
		log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
		for i := 0; i < maxRetries; i++ {
			_, err = s.t.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for endpoints to be created")
		}
		if err != nil {
			return err
		}

		serverAddr, err := s.getURLs(baseIngress)
		if err != nil {
			return err
		}

		_, err = s.t.KubeClient.CoreV1().ServiceAccounts(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		_, err = s.t.KubeClient.RbacV1beta1().Roles(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		_, err = s.t.KubeClient.RbacV1beta1().RoleBindings(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		time.Sleep(time.Second * 60)
		log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
		// Check Non DNS redirect
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
			}
			if resp.Status != 200 {
				return errors.New().WithMessage("Code did not matched").Err()
			}
		}

	}
	return nil
}

const (
	// Following is a fake SSL certificate data, generated for test purposes only.
	fakeHTTPAppsCodeDevCert = `-----BEGIN CERTIFICATE-----
MIIDCzCCAfOgAwIBAgIJAOaXTnfalwyQMA0GCSqGSIb3DQEBBQUAMBwxGjAYBgNV
BAMMEWh0dHAuYXBwc2NvZGUuZGV2MB4XDTE3MDcxODA5MTA0MFoXDTI3MDcxNjA5
MTA0MFowHDEaMBgGA1UEAwwRaHR0cC5hcHBzY29kZS5kZXYwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQCXn+4cxYbkFJ8qHrqORMPJ8a6/OtJooAwlsPWU
79z0kZ6RjBpw+hRRQvAxG4WPIpzqlhJcKAkQMOd5YlRZdoWi5P/fX+L5l8d2t1Yj
FnON/gZRvAX7alSvUBRdBFdZ/OJ6lDvVTWC+wYUnieePEmOnkd+ZopIaArLUEc3I
GljJRUG62srouOmTfbeCKdW5sI5R2UOo1pdrcxPN/J2lY6ixt8kneK80bosfpozu
9iVljWa7sO1s0Xsc/SwikDAIju8txpHEDl5SHcDX3JpVuNt9eeCquSuDNuegvjcH
RWzu/wHkcE7WGad7VkyXnzq1jBwBjryWINk3nzpmP7Q1BfnLAgMBAAGjUDBOMB0G
A1UdDgQWBBT08RnU4J5LD145GKdyMeRoWemOMjAfBgNVHSMEGDAWgBT08RnU4J5L
D145GKdyMeRoWemOMjAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBBQUAA4IBAQAv
pZFipxB65fuCZ4fV29Skxl4RwLWsvKRcKL7Fs+EyGhEF84B93N2jvwSO/fiifuHj
Q9algmNyftvEK5toHNIuGVSW35GpTGQ1GzNWlItlM5mmmXOK6kDvS8Yx4hszl8bz
ErhiVFmYp+huT7hI389VF5AIJ4Iqj6v0f1LKGa7jD2dJacFYWaHVV/z4W4LLvmKS
dxVm+Uu0HmX8D0vl+v2MHP/s7T20sx+VNcaw63HXeFmyn+EIa152jL1f12h2pB4t
4DZx5x7bvvGhTu/RktFl0rvT9vFkEOlmoy+ky4NlUDwyfLsRtXplQ2ltoyKvLge4
CstLLbiwGhfuzOGrsSD6
-----END CERTIFICATE-----
`
	fakeHTTPAppsCodeDevKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAl5/uHMWG5BSfKh66jkTDyfGuvzrSaKAMJbD1lO/c9JGekYwa
cPoUUULwMRuFjyKc6pYSXCgJEDDneWJUWXaFouT/31/i+ZfHdrdWIxZzjf4GUbwF
+2pUr1AUXQRXWfziepQ71U1gvsGFJ4nnjxJjp5HfmaKSGgKy1BHNyBpYyUVButrK
6Ljpk323ginVubCOUdlDqNaXa3MTzfydpWOosbfJJ3ivNG6LH6aM7vYlZY1mu7Dt
bNF7HP0sIpAwCI7vLcaRxA5eUh3A19yaVbjbfXngqrkrgzbnoL43B0Vs7v8B5HBO
1hmne1ZMl586tYwcAY68liDZN586Zj+0NQX5ywIDAQABAoIBABB/g244A/xvTf5M
R6pRSyh/Fq+SG/DscUXsolwpWVZ3PdTCdOIUI//Pk8kUII05i+9ukuLaLFpJp/Yq
P9lYLyRRXJIWoeDcpgSB4GqC9+HcYR2lotT/deV5hi202jhdbts9o+EKwVsgPXfW
5o5HxvYlxjm2WcVgw8qVgVmjnEOSDbvDgjb7yuCk9J2zIkYq+Qia+AzHqnIT4JmM
ZR+uxyQvhgwJQxXKMi/OqXL8AT4As8xBQb7L1FMXhmomyO1KAIz58DYUf+VCIXk1
S0Ama6sDg2yuJdDAk0mwFiJTlWbs4rzsBK9A9nGFZqCBGpCf7yjcuYhEk3giOoT8
qkszUIkCgYEAxcu89j2e+7/EUmAx6COtnWjFy3vx+JBMc8Hv1guze+qe4fTS7cvA
k3EHNjie+xXO2ZVfpGxpWFpUH/EH3Lo7dJfdPBRgqF9wSdpVlKVnQys6zw/aG5Ep
fEM2/NCBHDnWWqzW78/7I4/GSx0pVG5W8PkObv5vcCPUa9sclW+09nUCgYEAxD4N
EP93Drs19REIaCwZTJz4BMRmSCHA+Bfu0LdPEqTloVEv21zJZUiQt+e41wYwJRQK
7AUNl7leJJS3R34KCLZ9oRMhfOBU+2A5SHtg7j/Sx6UVCZhKFpjSJ/992qbJ4+4o
RASEMZ71WFKoVgHnT0Nhc4C2oBX+MQtT+C77pz8CgYBUdHTfs1oB5lTeU4Kbuzgz
YPwrsWWVG4/5UVKl02M0wu5KTq4NqRU2H2nT5gND9IDY+OXYoA2vEwqehN01izM9
ymZFc/H9kpqwfhBSovlffcLjjMI1SRssmsqM0j5+ndd/6hLwXJ7ABXDGu9Hc4iwv
Qji+fdd5S2M1Fl6zE/pxzQKBgQC0DH5uhwTUFj3GMC93bGZ13VrM/Oke6yEiPssU
4eqBn5szq8ptyC7bZ32nzcnQNtQ7YK04qNY0y5UtmOijhmdsYQrYmzXRXf16eWl1
MAXZ8eLQ24x2tivbmbDPk+EDmJ2JK3v0E/S5li9iLsxVxP9VwOuLTp/ANw12L/+F
qI2pfwKBgQCIJL+ltvMR1C75w2cW3v4xkC4fiV+kJ7GA0JMTftk9hws6iA620iWn
ciT4Bql5vJwULP7Sv+xLYK0tqnBE2dOzW23eAI5ZIlYiKDM9GGrRQvKIQmdRXSf1
oZmB+LUUEBO0+1+4QHcpbVlJlDLsv8cqcnLFpio4q+pFiAtuwq/G6w==
-----END RSA PRIVATE KEY-----`
)

func (s *IngressTestSuit) TestIngressSecretChanged() error {
	if s.t.Config.ProviderName == "minikube" {
		baseIngress := &api.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testIngressName(),
				Namespace: s.t.Config.TestNamespace,
			},
			Spec: api.IngressSpec{
				Rules: []api.IngressRule{
					{
						Host: "http.appscode.dev",
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.IngressBackend{
											ServiceName: testServerSvc.Name,
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		defer func() {
			if s.t.Config.Cleanup {
				s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		var svc *apiv1.Service
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for service to be created")
		}
		if err != nil {
			return errors.FromErr(err).Err()
		}

		// Check if svc have opened the port 80 for the ingress.
		if len(svc.Spec.Ports) != 1 {
			return errors.New("Service Port count didn't matched").Err()
		}

		if svc.Spec.Ports[0].Port != 80 {
			return errors.New("Service Port didn't matched").Err()
		}

		// Create and add a secret
		secret := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: s.t.Config.TestNamespace,
			},
			Type: apiv1.SecretTypeTLS,
			StringData: map[string]string{
				"tls.key": fakeHTTPAppsCodeDevKey,
				"tls.crt": fakeHTTPAppsCodeDevCert,
			},
		}
		_, err = s.t.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			return errors.FromErr(err).Err()
		}

		ing, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Get(baseIngress.Name)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		ing.Spec.TLS = []api.IngressTLS{{SecretName: secret.Name, Hosts: []string{baseIngress.Spec.Rules[0].Host}}}
		_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Update(ing)
		if err != nil {
			return errors.FromErr(err).Err()
		}

		time.Sleep(time.Second * 120)
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for service to be created")
		}
		if err != nil {
			return errors.FromErr(err).Err()
		}

		// Check if svc have opened the port 80 for the ingress.
		if len(svc.Spec.Ports) != 1 {
			return errors.New("Service Port count didn't matched").Err()
		}

		if svc.Spec.Ports[0].Port != 443 {
			return errors.New("Service Port didn't matched").Err()
		}
	}
	return nil
}
*/
