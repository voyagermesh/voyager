package e2e

import (
	"net/http"
	"text/template"
	"time"

	"github.com/appscode/errors"
	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	"k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const maxRetries = 50

var TestNamespace = "default"

var (
	defaultUrlTemplate = template.Must(template.New("svc-template").Parse("http://{{.IP}}:{{.Port}}"))
)

func (i *IngressTestSuit) TestIngressEnsureTPR() error {
	var err error
	for it := 0; it < 10; it++ {
		log.Infoln(it, "Trying to get ingress.appscode.com")
		tpr, err := i.t.KubeClient.Extensions().ThirdPartyResources().Get("ingress.appscode.com")
		if err == nil {
			log.Infoln("Found tpr for ingress with name", tpr.Name)
			break
		}
		err = errors.New().WithCause(err).Internal()
		time.Sleep(time.Second * 5)
		continue
	}
	return err
}

func (ing *IngressTestSuit) TestIngressCreate() error {
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: aci.ExtendedIngressBackend{
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

	_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(100)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}

	return nil
}

func (ing *IngressTestSuit) TestIngressDaemonCreate() error {
	if !ing.t.Config.InCluster && ing.t.Config.ProviderName != "minikube" {
		log.Infoln("Test is Running from outside of cluster skipping test")
		return nil
	}

	var nodeSelector = func() string {
		if ing.t.Config.ProviderName == "minikube" {
			return "kubernetes.io/hostname=minikube"
		} else {
			if len(ing.t.Config.DaemonHostName) > 0 {
				return "kubernetes.io/hostname=" + ing.t.Config.DaemonHostName
			}
			return "kubernetes.io/hostname=" + ing.t.Config.ClusterName + "-master"
		}
		return ""
	}

	baseDaemonIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
			Annotations: map[string]string{
				ingress.LBType:             ingress.LBDaemon,
				ingress.DaemonNodeSelector: nodeSelector(),
			},
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: aci.ExtendedIngressBackend{
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

	_, err := ing.t.ExtensionClient.Ingress(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := ing.t.KubeClient.Core().Services(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	serverAddr, err := ing.getDaemonURLs(baseDaemonIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints for test, Total url found", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressDelete() error {
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: aci.ExtendedIngressBackend{
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

	_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}

	err = ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).WithMessage("Failed to delete").Internal()
	}

	// Wait until everything is deleted
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
		if err != nil {
			if k8serr.IsNotFound(err) {
				break
			}
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be Deleted")
	}

	newServerAddr, err := ing.getURLs(baseIngress)
	if err == nil {
		if len(newServerAddr) != 0 {
			return errors.New().WithMessage("Server address found").Internal()
		}
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressUpdate() error {
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: aci.ExtendedIngressBackend{
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

	_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}

	updatedBaseIngress, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	updatedBaseIngress.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestpath"
	_, err = ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Update(updatedBaseIngress)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	time.Sleep(time.Second * 30)
	serverAddr, err = ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 20)
	log.Infoln("Loadbalancer created, calling http endpoints for updated path, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(1)
		if err == nil {
			return errors.New().WithCause(err).WithMessage("Connected with old prefix").Internal()
		}
		log.Infoln("Expected exception, faild to connect with old path, calling new paths.")
		resp, err = testserverclient.NewTestHTTPClient(url).Method("GET").Path("/newTestpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to Connect With New Prefix").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/newTestpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressCreateIPPersist() error {
	if len(ing.t.Config.LBPersistIP) > 0 &&
		(ing.t.Config.ProviderName == "gce" ||
			ing.t.Config.ProviderName == "gke" ||
			(ing.t.Config.ProviderName == "aws" && ing.t.Config.InCluster)) {
		baseIngress := &aci.Ingress{
			ObjectMeta: api.ObjectMeta{
				Name:      testIngressName(),
				Namespace: TestNamespace,
				Annotations: map[string]string{
					ingress.LoadBalancerPersist: "true",
					ingress.LoadBalancerIP:      ing.t.Config.LBPersistIP,
				},
			},
			Spec: aci.ExtendedIngressSpec{
				Rules: []aci.ExtendedIngressRule{
					{
						ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
							HTTP: &aci.HTTPExtendedIngressRuleValue{
								Paths: []aci.HTTPExtendedIngressPath{
									{
										Path: "/testpath",
										Backend: aci.ExtendedIngressBackend{
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

		_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return err
		}
		defer func() {
			if ing.t.Config.Cleanup {
				ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		var svc *api.Service
		for i := 0; i < maxRetries; i++ {
			svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
			_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for endpoints to be created")
		}
		if err != nil {
			return err
		}

		serverAddr, err := ing.getURLs(baseIngress)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 30)
		log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
			}
			log.Infoln("Response", *resp)
			if resp.Method != http.MethodGet {
				return errors.New().WithMessage("Method did not matched").Internal()
			}

			if resp.Path != "/testpath/ok" {
				return errors.New().WithMessage("Path did not matched").Internal()
			}
		}

		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		oldServiceIP := svc.Status.LoadBalancer.Ingress[0].IP

		err = ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}

		time.Sleep(time.Second * 30)
		baseIngress = &aci.Ingress{
			ObjectMeta: api.ObjectMeta{
				Name:      testIngressName(),
				Namespace: TestNamespace,
				Annotations: map[string]string{
					ingress.LoadBalancerPersist: "true",
					ingress.LoadBalancerIP:      oldServiceIP,
				},
			},
			Spec: aci.ExtendedIngressSpec{
				Rules: []aci.ExtendedIngressRule{
					{
						ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
							HTTP: &aci.HTTPExtendedIngressRuleValue{
								Paths: []aci.HTTPExtendedIngressPath{
									{
										Path: "/testpath",
										Backend: aci.ExtendedIngressBackend{
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

		_, err = ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
		if err != nil {
			return err
		}
		defer func() {
			if ing.t.Config.Cleanup {
				ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
			}
		}()

		// Wait sometime to loadbalancer be opened up.
		time.Sleep(time.Second * 10)
		for i := 0; i < maxRetries; i++ {
			svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
			_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for endpoints to be created")
		}
		if err != nil {
			return err
		}

		serverAddr, err = ing.getURLs(baseIngress)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 30)
		log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
		for _, url := range serverAddr {
			resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
			if err != nil {
				return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
			}
			log.Infoln("Response", *resp)
			if resp.Method != http.MethodGet {
				return errors.New().WithMessage("Method did not matched").Internal()
			}

			if resp.Path != "/testpath/ok" {
				return errors.New().WithMessage("Path did not matched").Internal()
			}
		}

		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
		if err != nil {
			return errors.New().WithCause(err).Internal()
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
			return errors.New().WithMessage("Service Ip not matched with previous IP").Internal()
		}
	} else {
		log.Infoln("Test Skipped")
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressCreateWithOptions() error {
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Backend: aci.ExtendedIngressBackend{
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

	_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/override/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}

		if resp.Headers.Get("X-Ingress-Test-Header") != "ingress.appscode.com" {
			return errors.New().WithMessage("Header did not matched").Internal()
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
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/override/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}

		if resp.Headers.Get("X-Ingress-Test-Header") != "ingress.appscode.com/v1beta1" {
			return errors.New().WithMessage("Header did not matched").Internal()
		}
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressCoreIngress() error {
	baseIngress := &extensions.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: TestNamespace,
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

	_, err := ing.t.KubeClient.Extensions().Ingresses(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.KubeClient.Extensions().Ingresses(baseIngress.Namespace).Delete(baseIngress.Name, &api.DeleteOptions{})
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	baseExtIngress, err := aci.NewEngressFromIngress(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	serverAddr, err := ing.getURLs(baseExtIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
	log.Infoln("Loadbalancer created, calling http endpoints, Total", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").DoWithRetry(50)
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}
	return nil
}
