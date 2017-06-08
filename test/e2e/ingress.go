package e2e

import (
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	api "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const maxRetries = 50

var (
	defaultUrlTemplate = template.Must(template.New("svc-template").Parse("http://{{.IP}}:{{.Port}}"))
)

func (s *IngressTestSuit) TestIngressEnsureTPR() error {
	var err error
	for it := 0; it < 10; it++ {
		log.Infoln(it, "Trying to get ingress.appscode.com")
		tpr, err := s.t.KubeClient.Extensions().ThirdPartyResources().Get("ingress.appscode.com")
		if err == nil {
			log.Infoln("Found tpr for ingress with name", tpr.Name)
			break
		}
		err = errors.New().WithCause(err).Err()
		time.Sleep(time.Second * 5)
		continue
	}
	return err
}

func (s *IngressTestSuit) TestIngressCreateDelete() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(100)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}

	if s.t.Voyager != nil && s.t.Voyager.ProviderName != "minikube" {
		// Check Status for ingress
		baseIngress, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			if len(baseIngress.Status.LoadBalancer.Ingress) != len(svc.Status.LoadBalancer.Ingress) {
				return errors.New().WithMessage("Statuses didn't matched").Err()
			}
			if baseIngress.Status.LoadBalancer.Ingress[0] != svc.Status.LoadBalancer.Ingress[0] {
				return errors.New().WithMessage("Statuses didn't matched").Err()
			}
		}
	}

	err = s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Failed to delete").Err()
	}

	// Wait until everything is deleted
	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err != nil {
			if k8serr.IsNotFound(err) {
				break
			}
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be Deleted")
	}
	if !k8serr.IsNotFound(err) {
		return errors.New().WithCause(err).Err()
	}
	return nil
}

func (s *IngressTestSuit) TestIngressUpdate() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 40)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}

	updatedBaseIngress, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	updatedBaseIngress.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestpath"
	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Update(updatedBaseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 30)
	serverAddr, err = s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 60)
	log.Infoln("Loadbalancer created, calling http endpoints for updated path, Total", len(serverAddr))
	for _, url := range serverAddr {
		var resp *testserverclient.Response
		notFound := false
		for i := 0; i < maxRetries; i++ {
			var err error
			resp, err = testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(1)
			if err != nil {
				notFound = true
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Expected exception, faild to connect with old path, calling new paths.")
		}
		if !notFound {
			return errors.New().WithCause(err).WithMessage("Connected with old prefix").Err()
		}
		resp, err = testserverclient.NewTestHTTPClient(url).Method("GET").Path("/newTestpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to Connect With New Prefix").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/newTestpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}

	// Open New Port
	updatedBaseIngress, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if s.t.Config.ProviderName != "minikube" {
		updatedBaseIngress.Spec.Rules[0].HTTP = nil
		updatedBaseIngress.Spec.Rules[0].TCP = []api.TCPExtendedIngressRuleValue{
			{
				Port: intstr.FromString("4545"),
				Backend: api.IngressBackend{
					ServiceName: testServerSvc.Name,
					ServicePort: intstr.FromString("4545"),
				},
			},
			{
				Port: intstr.FromString("4949"),
				Backend: api.IngressBackend{
					ServiceName: testServerSvc.Name,
					ServicePort: intstr.FromString("4545"),
				},
			},
		}
		_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Update(updatedBaseIngress)
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
		time.Sleep(time.Second * 30)

		found := false
		for i := 1; i <= maxRetries; i++ {
			svc, err := s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
			if err != nil {
				continue
			}
			log.Infoln("Got Service", svc.Name)
			for _, port := range svc.Spec.Ports {
				log.Infoln(port)
				if port.Port == 4545 {
					found = true
					break
				}
			}
			if found {
				break
			}
			time.Sleep(time.Second * 5)
		}

		if !found {
			return errors.New().WithMessage("Service not found").Err()
		}

		serverAddr, err = s.getURLs(baseIngress)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 30)
		log.Infoln("Loadbalancer created, calling http endpoints for updated path, Total", len(serverAddr))
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestTCPClient(url).DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to Connect With New Prefix").Err()
			}
			log.Infoln("Response", *resp)
			if resp.ServerPort != ":4545" {
				return errors.New().WithMessage("Port did not matched").Err()
			}
		}

		log.Infoln("Checking NodePort Assignments")
		rc, err := s.t.KubeClient.Core().ReplicationControllers(s.t.Config.TestNamespace).Get(testServerRc.Name)
		if err == nil {

			svc, err := s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
			if err != nil {
				return errors.New().WithMessage("Service get encountered error").Err()
			}
			// Removing pods so that endpoints get updated
			rc.Spec.Replicas = 0
			s.t.KubeClient.Core().ReplicationControllers(s.t.Config.TestNamespace).Update(rc)

			for {
				pods, _ := s.t.KubeClient.Core().Pods(s.t.Config.TestNamespace).List(kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(rc.Spec.Selector)),
				})
				if len(pods.Items) <= 0 {
					break
				}
				time.Sleep(time.Second * 5)
			}
			svcUpdated, err := s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
			if err != nil {
				return errors.New().WithMessage("Service get encountered error").Err()
			}

			for _, oldPort := range svc.Spec.Ports {
				for _, newPort := range svcUpdated.Spec.Ports {
					if oldPort.Port == newPort.Port {
						if oldPort.NodePort != newPort.NodePort {
							return errors.New().WithMessage("Node Port Mismatched").Err()
						}
					}
				}
			}

			rc.Spec.Replicas = 2
			s.t.KubeClient.Core().ReplicationControllers(s.t.Config.TestNamespace).Update(rc)

			svcUpdated, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
			if err != nil {
				return errors.New().WithMessage("Service get encountered error").Err()
			}

			for _, oldPort := range svc.Spec.Ports {
				for _, newPort := range svcUpdated.Spec.Ports {
					if oldPort.Port == newPort.Port {
						if oldPort.NodePort != newPort.NodePort {
							return errors.New().WithMessage("Node Port Mismatched").Err()
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressCreateIPPersist() error {
	if len(s.t.Config.LBPersistIP) > 0 &&
		(s.t.Config.ProviderName == "gce" ||
			s.t.Config.ProviderName == "gke" ||
			(s.t.Config.ProviderName == "aws" && s.t.Config.InCluster)) {
		baseIngress := &api.Ingress{
			ObjectMeta: kapi.ObjectMeta{
				Name:      testIngressName(),
				Namespace: s.t.Config.TestNamespace,
				Annotations: map[string]string{
					api.LoadBalancerPersist: s.t.Config.LBPersistIP,
				},
			},
			Spec: api.ExtendedIngressSpec{
				Rules: []api.ExtendedIngressRule{
					{
						ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
							HTTP: &api.HTTPExtendedIngressRuleValue{
								Paths: []api.HTTPExtendedIngressPath{
									{
										Path: "/testpath",
										Backend: api.ExtendedIngressBackend{
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

		_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return err
		}
		defer func() {
			if s.t.Config.Cleanup {
				s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		var svc *kapi.Service
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
			_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		time.Sleep(time.Second * 30)
		log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
			}
			log.Infoln("Response", *resp)
			if resp.Method != http.MethodGet {
				return errors.New().WithMessage("Method did not matched").Err()
			}

			if resp.Path != "/testpath/ok" {
				return errors.New().WithMessage("Path did not matched").Err()
			}
		}

		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
		oldServiceIP := svc.Status.LoadBalancer.Ingress[0].IP

		err = s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		if err != nil {
			return errors.New().WithCause(err).Err()
		}

		time.Sleep(time.Second * 30)
		baseIngress = &api.Ingress{
			ObjectMeta: kapi.ObjectMeta{
				Name:      testIngressName(),
				Namespace: s.t.Config.TestNamespace,
				Annotations: map[string]string{
					api.LoadBalancerPersist: oldServiceIP,
				},
			},
			Spec: api.ExtendedIngressSpec{
				Rules: []api.ExtendedIngressRule{
					{
						ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
							HTTP: &api.HTTPExtendedIngressRuleValue{
								Paths: []api.HTTPExtendedIngressPath{
									{
										Path: "/testpath",
										Backend: api.ExtendedIngressBackend{
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

		_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return err
		}
		defer func() {
			if s.t.Config.Cleanup {
				s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		for i := 0; i < maxRetries; i++ {
			svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
			_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		time.Sleep(time.Second * 30)
		log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
			}
			log.Infoln("Response", *resp)
			if resp.Method != http.MethodGet {
				return errors.New().WithMessage("Method did not matched").Err()
			}

			if resp.Path != "/testpath/ok" {
				return errors.New().WithMessage("Path did not matched").Err()
			}
		}

		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err != nil {
			return errors.New().WithCause(err).Err()
		}

		found := false
		for _, lbIngress := range svc.Status.LoadBalancer.Ingress {
			log.Infoln("Matching Service Ips for Ingress", lbIngress.IP, oldServiceIP)
			if lbIngress.IP == oldServiceIP {
				found = true
				break
			}
		}

		if !found {
			log.Infoln("Service Ip not matched with previous IP")
			return errors.New().WithMessage("Service Ip not matched with previous IP").Err()
		}
	} else {
		log.Infoln("Test Skipped")
	}
	return nil
}

func (s *IngressTestSuit) TestIngressCreateWithOptions() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Backend: api.ExtendedIngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
										HeaderRule: []string{
											"X-Ingress-Test-Header ingress.appscode.com",
										},
										RewriteRule: []string{
											`^([^\ :]*)\ /(.*)$ \1\ /override/\2`,
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/override/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}

		if resp.RequestHeaders.Get("X-Ingress-Test-Header") != "ingress.appscode.com" {
			return errors.New().WithMessage("Header did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).
			Method("GET").
			Header(map[string]string{
				"X-Ingress-Test-Header": "ingress.appscode.com/v1beta1",
			}).
			Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/override/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}

		if resp.RequestHeaders.Get("X-Ingress-Test-Header") != "ingress.appscode.com/v1beta1" {
			return errors.New().WithMessage("Header did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressCoreIngress() error {
	baseIngress := &extensions.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "voyager",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				{
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: extensions.IngressBackend{
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

	_, err := s.t.KubeClient.Extensions().Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.Extensions().Ingresses(baseIngress.Namespace).Delete(baseIngress.Name, &kapi.DeleteOptions{})
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(api.VoyagerPrefix + baseIngress.Name)
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	baseExtIngress, err := api.NewEngressFromIngress(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	serverAddr, err := s.getURLs(baseExtIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressHostNames() error {
	headlessSvc, err := s.t.KubeClient.Core().Services(s.t.Config.TestNamespace).Create(testStatefulSetSvc)
	if err != nil {
		return err
	}
	orphan := false
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.Core().Services(s.t.Config.TestNamespace).Delete(headlessSvc.Name, &kapi.DeleteOptions{
				OrphanDependents: &orphan,
			})
		}
	}()

	ss, err := s.t.KubeClient.Apps().StatefulSets(s.t.Config.TestNamespace).Create(testServerStatefulSet)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.Apps().StatefulSets(s.t.Config.TestNamespace).Delete(ss.Name, &kapi.DeleteOptions{
				OrphanDependents: &orphan,
			})
		}
	}()

	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
										HostNames:   []string{testServerStatefulSet.Name + "-0"},
										ServiceName: headlessSvc.Name,
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
	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 120)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath").DoWithRetry(100)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}
		if resp.PodName != ss.Name+"-0" {
			return errors.New().WithMessage("PodName did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressBackendWeight() error {
	dp1, err := s.t.KubeClient.Extensions().Deployments(s.t.Config.TestNamespace).Create(&extensions.Deployment{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "deploymet-1-" + randString(4),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v1",
				},
			},
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v1",
					},
					Annotations: map[string]string{
						api.BackendWeight: "90",
					},
				},
				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:  "server",
							Image: "appscode/test-server:1.1",
							Env: []kapi.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &kapi.EnvVarSource{
										FieldRef: &kapi.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []kapi.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	dp2, err := s.t.KubeClient.Extensions().Deployments(s.t.Config.TestNamespace).Create(&extensions.Deployment{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "deploymet-2-" + randString(4),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "deployment",
					"app-version": "v2",
				},
			},
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: map[string]string{
						"app":         "deployment",
						"app-version": "v2",
					},
					Annotations: map[string]string{
						api.BackendWeight: "10",
					},
				},
				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:  "server",
							Image: "appscode/test-server:1.1",
							Env: []kapi.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &kapi.EnvVarSource{
										FieldRef: &kapi.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []kapi.ContainerPort{
								{
									Name:          "http-1",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	svc, err := s.t.KubeClient.Core().Services(s.t.Config.TestNamespace).Create(&kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "deployment-svc",
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{
					Name:       "http-1",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   "TCP",
				},
			},
			Selector: map[string]string{
				"app": "deployment",
			},
		},
	})
	if err != nil {
		return err
	}

	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
										ServiceName: svc.Name,
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

	defer func() {
		if s.t.Config.Cleanup {
			dp1, err := s.t.KubeClient.Extensions().Deployments(dp1.Namespace).Get(dp1.Name)
			if err == nil {
				dp1.Spec.Replicas = 0
				s.t.KubeClient.Extensions().Deployments(dp1.Namespace).Update(dp1)
			}
			dp2, err := s.t.KubeClient.Extensions().Deployments(dp2.Namespace).Get(dp2.Name)
			if err == nil {
				dp2.Spec.Replicas = 0
				s.t.KubeClient.Extensions().Deployments(dp2.Namespace).Update(dp2)
			}
			time.Sleep(time.Second * 5)
			orphan := false
			s.t.KubeClient.Extensions().Deployments(dp1.Namespace).Delete(dp1.Name, &kapi.DeleteOptions{
				OrphanDependents: &orphan,
			})

			s.t.KubeClient.Extensions().Deployments(dp2.Namespace).Delete(dp2.Name, &kapi.DeleteOptions{
				OrphanDependents: &orphan,
			})

			s.t.KubeClient.Core().Services(svc.Namespace).Delete(svc.Name, &kapi.DeleteOptions{
				OrphanDependents: &orphan,
			})
		}
	}()

	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	serverAddr, err := s.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints for test, Total url found", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}
	var deploymentCounter1, deploymentCounter2 int
	for cnt := 0; cnt < 100; cnt++ {
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
			}
			log.Infoln("Response", *resp)
			if resp.Method != http.MethodGet {
				return errors.New().WithMessage("Method did not matched").Err()
			}

			if strings.HasPrefix(resp.PodName, dp1.Name) {
				deploymentCounter1++
			} else if strings.HasPrefix(resp.PodName, dp2.Name) {
				deploymentCounter2++
			}
		}
	}

	if deploymentCounter2 != 10 {
		return errors.New().WithMessagef("Expected %v Actual %v", 10, deploymentCounter2).Err()
	}

	if deploymentCounter1 != 90 {
		return errors.New().WithMessagef("Expected %v Actual %v", 90, deploymentCounter1).Err()
	}
	return nil
}

func (s *IngressTestSuit) TestIngressBackendRule() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/old",
									Backend: api.ExtendedIngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
										BackendRule: []string{
											"acl add_url capture.req.uri -m beg /old/add/now",
											`http-response set-header X-Added-From-Proxy added-from-proxy if add_url`,

											"acl rep_url path_beg /old/replace",
											`reqrep ^([^\ :]*)\ /(.*)$ \1\ /rewrited/from/proxy/\2 if rep_url`,
										},
									},
								},
								{
									Path: "/test-second",
									Backend: api.ExtendedIngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
										BackendRule: []string{
											"acl add_url capture.req.uri -m beg /test-second",
											`http-response set-header X-Added-From-Proxy added-from-proxy if add_url`,

											"acl rep_url path_beg /test-second",
											`reqrep ^([^\ :]*)\ /(.*)$ \1\ /rewrited/from/proxy/\2 if rep_url`,
										},
										HeaderRule: []string{
											"X-Ingress-Test-Header ingress.appscode.com",
										},
										RewriteRule: []string{
											`^([^\ :]*)\ /(.*)$ \1\ /override/\2`,
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/old/replace").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/rewrited/from/proxy/old/replace" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/old/add/now").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Err()
		}

		if resp.Path != "/old/add/now" {
			return errors.New().WithMessage("Path did not matched").Err()
		}

		if resp.ResponseHeader.Get("X-Added-From-Proxy") != "added-from-proxy" {
			return errors.New().WithMessage("Header did not matched").Err()
		}
	}

	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/test-second").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Metho/replaced did not matched").Err()
		}

		if resp.RequestHeaders.Get("X-Ingress-Test-Header") != "ingress.appscode.com" {
			return errors.New().WithMessage("Header did not matched").Err()
		}

		if resp.ResponseHeader.Get("X-Added-From-Proxy") != "added-from-proxy" {
			return errors.New().WithMessage("Header did not matched").Err()
		}

		if resp.RequestHeaders.Get("X-Ingress-Test-Header") != "ingress.appscode.com" {
			return errors.New().WithMessage("Header did not matched").Err()
		}

		if resp.Path != "/override/rewrited/from/proxy/test-second" {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressAnnotations() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.ServiceAnnotations: `{"foo": "bar", "service-annotation": "set"}`,
				api.PodAnnotations:     `{"foo": "bar", "pod-annotation": "set"}`,
			},
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if svc.Annotations == nil {
		return errors.New().WithMessage("Service annotations nil").Err()
	}

	if val := svc.Annotations["foo"]; val != "bar" {
		return errors.New().WithMessage("Service annotations didn't matched").Err()
	}

	if val := svc.Annotations["service-annotation"]; val != "set" {
		return errors.New().WithMessage("Service annotations didn't matched").Err()
	}

	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	pods, err := s.t.KubeClient.Core().Pods(svc.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector),
	})
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Annotations == nil {
				return errors.New().WithCause(err).WithMessagef("Pods %s annotations nil", pod.Name).Err()
			}

			if val := pod.Annotations["foo"]; val != "bar" {
				return errors.New().WithMessage("Service annotations didn't matched").Err()
			}

			if val := pod.Annotations["pod-annotation"]; val != "set" {
				return errors.New().WithMessage("Service annotations didn't matched").Err()
			}
		}
	}

	// Check Service Annotation Change only Update Service
	ings, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Ingress error").Err()
	}
	ings.Annotations[api.ServiceAnnotations] = `{"bar": "foo", "second-service-annotation": "set"}`
	ings, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Update(ings)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Ingress error").Err()
	}

	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err != nil {
			err = errors.New().WithCause(err).WithMessage("Service encountered an error").Err()
		}
		if svc.Annotations == nil {
			err = errors.New().WithMessage("Service annotations nil").Err()
		}
		if _, ok := svc.Annotations["foo"]; ok {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
		}
		if val := svc.Annotations["second-service-annotation"]; val != "set" {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
		}
		if val := svc.Annotations["bar"]; val != "foo" {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
		}
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		return errors.FromErr(err).Err()
	}

	pods, err = s.t.KubeClient.Core().Pods(svc.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector),
	})
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Annotations == nil {
				return errors.New().WithCause(err).WithMessagef("Pods %s annotations nil", pod.Name).Err()
			}

			if val := pod.Annotations["foo"]; val != "bar" {
				return errors.New().WithMessage("Pod annotations didn't matched").Err()
			}

			if val := pod.Annotations["pod-annotation"]; val != "set" {
				return errors.New().WithMessage("Pod annotations didn't matched").Err()
			}
		}
	}

	// Check Pod Annotation Change only Update Pods
	ings, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Ingress error").Err()
	}
	ings.Annotations[api.PodAnnotations] = `{"bar": "foo", "second-pod-annotation": "set"}`
	ings, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Update(ings)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Ingress error").Err()
	}

	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
		if err != nil {
			err = errors.New().WithCause(err).WithMessage("Service encountered an error").Err()
		}
		if svc.Annotations == nil {
			err = errors.New().WithMessage("Service annotations nil").Err()
		}
		if _, ok := svc.Annotations["foo"]; ok {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
		}
		if val := svc.Annotations["second-service-annotation"]; val != "set" {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
		}
		if val := svc.Annotations["bar"]; val != "foo" {
			err = errors.New().WithMessage("Service annotations didn't matched").Err()
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
		}
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}

	pods, err = s.t.KubeClient.Core().Pods(svc.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector),
	})
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Annotations == nil {
				return errors.New().WithCause(err).WithMessagef("Pods %s annotations nil", pod.Name).Err()
			}

			if _, ok := pod.Annotations["foo"]; ok {
				return errors.New().WithMessage("Pod annotations didn't matched").Err()
			}

			if val := pod.Annotations["bar"]; val != "foo" {
				return errors.New().WithMessage("Pod annotations didn't matched").Err()
			}

			if val := pod.Annotations["second-pod-annotation"]; val != "set" {
				return errors.New().WithMessage("Pod annotations didn't matched").Err()
			}
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressNodePort() error {
	baseDaemonIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.LBType: api.LBTypeNodePort,
			},
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	var svc *kapi.Service
	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		var err error
		svc, err = s.t.KubeClient.Core().Services(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if svc.Spec.Type != kapi.ServiceTypeNodePort {
		return errors.New().WithMessage("ServiceType Not NodePort").Err()
	}

	// We do not open any firewall for node ports, so we can not check the traffic
	// for testing. So check if all the ports are assigned a nodeport.
	time.Sleep(time.Second * 120)
	for _, port := range svc.Spec.Ports {
		if port.NodePort <= 0 {
			return errors.New().WithMessagef("NodePort not Assigned for Port %v -> %v", port.Port, port.NodePort).Err()
		}
	}

	return nil
}

func (s *IngressTestSuit) TestIngressStats() error {
	baseIng := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.StatsOn:   "true",
				api.StatsPort: "8787",
			},
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIng.Namespace).Create(baseIng)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIng.Namespace).Delete(baseIng.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		var err error
		_, err = s.t.KubeClient.Core().Services(baseIng.Namespace).Get(baseIng.OffshootName())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	// Check if all Stats Things are open
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		var err error
		svc, err = s.t.KubeClient.Core().Services(baseIng.Namespace).Get(baseIng.Name + "-stats")
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if svc.Spec.Ports[0].Port != 8787 {
		return errors.New().WithMessage("Service port mismatched").Err()
	}

	// Remove Stats From Annotation and Check if the service gets deleted
	baseIng, err = s.t.ExtClient.Ingress(baseIng.Namespace).Get(baseIng.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	delete(baseIng.Annotations, api.StatsOn)
	baseIng, err = s.t.ExtClient.Ingress(baseIng.Namespace).Update(baseIng)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 60)
	var deleteErr error
	for i := 0; i < maxRetries; i++ {
		_, deleteErr = s.t.KubeClient.Core().Services(baseIng.Namespace).Get(baseIng.Name + "-stats")
		if deleteErr != nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be Deleted")
	}
	if deleteErr == nil {
		return errors.New().WithMessage("Stats Service Should Be deleted").Err()
	}

	return nil
}

func (s *IngressTestSuit) TestIngressKeepSource() error {
	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.KeepSourceIP: "true",
			},
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.KeepSourceIP: "true",
			},
		},
		Spec: api.ExtendedIngressSpec{
			LoadBalancerSourceRanges: []string{
				"192.101.0.0/16",
				"192.0.0.0/24",
			},
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
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

	_, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 60)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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

	tobeUpdated, err := s.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	tobeUpdated.Spec.LoadBalancerSourceRanges = []string{"192.10.0.0/24"}
	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Update(tobeUpdated)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 60)
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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

func (s *IngressTestSuit) TestIngressExternalName() error {
	extSvc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "external-svc-non-dns",
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: kapi.ServiceSpec{
			Type:         kapi.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err := s.t.KubeClient.Core().Services(extSvc.Namespace).Create(extSvc)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.Core().Services(extSvc.Namespace).Delete(extSvc.Name, nil)
		}
	}()

	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
										ServiceName: extSvc.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath").DoTestRedirectWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 301 {
			return errors.New().WithMessage("Path did not matched").Err()
		}
		if resp.ResponseHeader.Get("Location") != "http://google.com" {
			return errors.New().WithMessage("Location did not matched").Err()
		}
	}
	return nil
}

func (s *IngressTestSuit) TestIngressExternalNameResolver() error {
	extSvc := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "external-svc-dns",
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.ExternalDNSResolvers: `{"nameserver": [{"mode": "dnsmasq", "address": "8.8.8.8:53"}]}`,
			},
		},
		Spec: kapi.ServiceSpec{
			Type:         kapi.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	_, err := s.t.KubeClient.Core().Services(extSvc.Namespace).Create(extSvc)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.KubeClient.Core().Services(extSvc.Namespace).Delete(extSvc.Name, nil)
		}
	}()

	baseIngress := &api.Ingress{
		ObjectMeta: kapi.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
		},
		Spec: api.ExtendedIngressSpec{
			Rules: []api.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: api.ExtendedIngressRuleValue{
						HTTP: &api.HTTPExtendedIngressRuleValue{
							Paths: []api.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: api.ExtendedIngressBackend{
										ServiceName: extSvc.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *kapi.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = s.t.KubeClient.Core().Services(baseIngress.Namespace).Get(baseIngress.OffshootName())
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
		_, err = s.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
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
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath").DoStatusWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Err()
		}
		if resp.Status != 404 {
			return errors.New().WithMessage("Path did not matched").Err()
		}
	}
	return nil
}
