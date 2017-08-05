package e2e

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	api "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func daemonNodeSelector(s *IngressTestSuit) string {
	if s.t.Config.ProviderName == "minikube" {
		return `{"kubernetes.io/hostname": "minikube"}`
	} else {
		if len(s.t.Config.DaemonHostName) > 0 {
			return fmt.Sprintf(`{"kubernetes.io/hostname": "%s"}`, s.t.Config.DaemonHostName)
		}
	}
	log.Warningln("No node selector provided for daemon ingress")
	return "{}"
}

// DaemonSet Tests can not run more than once at a time.
var daemonTestLock sync.Mutex

func (s *IngressTestSuit) TestIngressDaemonCreate() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

	if !s.t.Config.InCluster && s.t.Config.ProviderName != "minikube" {
		log.Infoln("Test is Running from outside of cluster skipping test")
		return nil
	}

	baseDaemonIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.LBType:       api.LBTypeHostPort,
				api.NodeSelector: daemonNodeSelector(s),
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

	_, err := s.t.ExtClient.Ingresses(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 30)
	for i := 0; i < maxRetries; i++ {
		_, err := s.t.KubeClient.CoreV1().Services(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	serverAddr, err := s.getDaemonURLs(baseDaemonIngress)
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

func (s *IngressTestSuit) TestIngressDaemonUpdate() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

	if !s.t.Config.InCluster && s.t.Config.ProviderName != "minikube" {
		log.Infoln("Test is Running from outside of cluster skipping test")
		return nil
	}

	baseIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.LBType:       api.LBTypeHostPort,
				api.NodeSelector: daemonNodeSelector(s),
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
		return errors.New().WithCause(err).Err()
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseIngress.Namespace).Delete(baseIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return err
	}

	serverAddr, err := s.getDaemonURLs(baseIngress)
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

	updatedBaseIngress, err := s.t.ExtClient.Ingresses(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}
	updatedBaseIngress.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestpath"
	_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Update(updatedBaseIngress)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	time.Sleep(time.Second * 30)
	serverAddr, err = s.getDaemonURLs(baseIngress)
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
	updatedBaseIngress, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Get(baseIngress.Name)
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	if s.t.Config.ProviderName != "minikube" {
		updatedBaseIngress.Spec.Rules[0].HTTP = nil
		updatedBaseIngress.Spec.Rules[0].TCP = &api.TCPIngressRuleValue{
			Port: intstr.FromString("4545"),
			Backend: api.IngressBackend{
				ServiceName: testServerSvc.Name,
				ServicePort: intstr.FromString("4545"),
			},
		}
		_, err = s.t.ExtClient.Ingresses(baseIngress.Namespace).Update(updatedBaseIngress)
		if err != nil {
			return errors.New().WithCause(err).Err()
		}
		time.Sleep(time.Second * 60)
		found := false
		for i := 1; i <= maxRetries; i++ {
			svc, err := s.t.KubeClient.CoreV1().Services(baseIngress.Namespace).Get(baseIngress.OffshootName(), metav1.GetOptions{})
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

		serverAddr, err = s.getDaemonURLs(baseIngress)
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

func (s *IngressTestSuit) TestIngressDaemonRestart() error {
	daemonTestLock.Lock()
	defer daemonTestLock.Unlock()

	if !s.t.Config.InCluster && s.t.Config.ProviderName != "minikube" {
		log.Infoln("Test is Running from outside of cluster skipping test")
		return nil
	}

	baseDaemonIngress := &api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressName(),
			Namespace: s.t.Config.TestNamespace,
			Annotations: map[string]string{
				api.LBType:       api.LBTypeHostPort,
				api.NodeSelector: daemonNodeSelector(s),
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

	_, err := s.t.ExtClient.Ingresses(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if s.t.Config.Cleanup {
			s.t.ExtClient.Ingresses(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := s.t.KubeClient.CoreV1().Services(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName(), metav1.GetOptions{})
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Err()
	}

	serverAddr, err := s.getDaemonURLs(baseDaemonIngress)
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
	_, err = s.t.KubeClient.ExtensionsV1beta1().DaemonSets(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	rc, err := s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas = types.Int32P(*rc.Spec.Replicas + 1)
	s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Update(rc)

	rc, err = s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas = types.Int32P(*rc.Spec.Replicas - 1)
	s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Update(rc)

	_, err = s.t.KubeClient.ExtensionsV1beta1().DaemonSets(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	rc, err = s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	rc.Spec.Replicas = types.Int32P(*rc.Spec.Replicas + 1)
	s.t.KubeClient.CoreV1().ReplicationControllers(testServerRc.Namespace).Update(rc)

	_, err = s.t.KubeClient.ExtensionsV1beta1().DaemonSets(baseDaemonIngress.Namespace).Get(baseDaemonIngress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}
