package e2e

import (
	"net/http"
	"sync"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

// DaemonSet Tests can not run more than once at a time.
var daemonTestLock sync.Mutex

func (ing *IngressTestSuit) TestIngressDaemonCreate() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

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
			Namespace: ing.t.Config.TestNamespace,
			Annotations: map[string]string{
				ingress.LBType:       ingress.LBTypeHostPort,
				ingress.NodeSelector: nodeSelector(),
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

	_, err := ing.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 30)
	for i := 0; i < maxRetries; i++ {
		_, err := ing.t.KubeClient.Core().Services(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
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

func (ing *IngressTestSuit) TestIngressDaemonUpdate() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

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
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      testIngressName(),
			Namespace: ing.t.Config.TestNamespace,
			Annotations: map[string]string{
				ingress.LBType:       ingress.LBTypeHostPort,
				ingress.NodeSelector: nodeSelector(),
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

	_, err := ing.t.ExtClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := ing.getDaemonURLs(baseIngress)
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

	updatedBaseIngress, err := ing.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	updatedBaseIngress.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestpath"
	_, err = ing.t.ExtClient.Ingress(baseIngress.Namespace).Update(updatedBaseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 30)
	serverAddr, err = ing.getDaemonURLs(baseIngress)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
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
		resp, err = testserverclient.NewTestHTTPClient(url).Method("GET").Path("/newTestpath/ok").DoWithRetry(100)
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
	updatedBaseIngress, err = ing.t.ExtClient.Ingress(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if ing.t.Config.ProviderName != "minikube" {
		updatedBaseIngress.Spec.Rules[0].HTTP = nil
		updatedBaseIngress.Spec.Rules[0].TCP = []aci.TCPExtendedIngressRuleValue{
			{
				Port: intstr.FromString("4545"),
				Backend: aci.IngressBackend{
					ServiceName: testServerSvc.Name,
					ServicePort: intstr.FromString("4545"),
				},
			},
		}
		_, err = ing.t.ExtClient.Ingress(baseIngress.Namespace).Update(updatedBaseIngress)
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
		time.Sleep(time.Second * 60)
		found := false
		for i := 1; i <= maxRetries; i++ {
			svc, err := ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
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

		serverAddr, err = ing.getDaemonURLs(baseIngress)
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
	}
	return nil
}

func (ing *IngressTestSuit) TestIngressDaemonRestart() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

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
			Namespace: ing.t.Config.TestNamespace,
			Annotations: map[string]string{
				ingress.LBType:       ingress.LBTypeHostPort,
				ingress.NodeSelector: nodeSelector(),
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

	_, err := ing.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.Config.Cleanup {
			ing.t.ExtClient.Ingress(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
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
		return errors.New().WithCause(err).Err()
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

	// Teardown and then again create one pod of the backend
	// And Make sure The DS does not gets deleted.
	_, err = ing.t.KubeClient.Extensions().DaemonSets(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
	if err != nil {
		return err
	}
	rc, err := ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas += 1
	ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Update(rc)

	rc, err = ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas -= 1
	ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Update(rc)

	_, err = ing.t.KubeClient.Extensions().DaemonSets(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	rc, err = ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas += 1
	ing.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Update(rc)

	_, err = ing.t.KubeClient.Extensions().DaemonSets(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}
