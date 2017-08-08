package framework

import (
	"errors"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/ingress"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	testServerImage = "appscode/test-server:1.4"
)

var (
	testServerResourceName = "e2e-test-server" + rand.Characters(5)
)

func (i *ingressInvocation) Setup() error {
	if err := i.setupTestServers(); err != nil {
		return err
	}
	return i.waitForTestServer()
}

func (i *ingressInvocation) Teardown() {
	if i.Config.Cleanup {
		i.KubeClient.CoreV1().Services(i.Namespace()).Delete(testServerResourceName, &metav1.DeleteOptions{})
		rc, err := i.KubeClient.CoreV1().ReplicationControllers(i.Namespace()).Get(testServerResourceName, metav1.GetOptions{})
		if err == nil {
			rc.Spec.Replicas = types.Int32P(0)
			i.KubeClient.CoreV1().ReplicationControllers(i.Namespace()).Update(rc)
			time.Sleep(time.Second * 5)
		}
		i.KubeClient.CoreV1().ReplicationControllers(i.Namespace()).Delete(testServerResourceName, &metav1.DeleteOptions{})
	}
}

func (i *ingressInvocation) TestServerName() string {
	return testServerResourceName
}

func (i *ingressInvocation) Create(ing *api.Ingress) error {
	_, err := i.VoyagerClient.Ingresses(i.Namespace()).Create(ing)
	if err != nil {
		return err
	}
	return nil
}

func (i *ingressInvocation) Get(ing *api.Ingress) (*api.Ingress, error) {
	return i.VoyagerClient.Ingresses(i.Namespace()).Get(ing.Name)
}

func (i *ingressInvocation) Update(ing *api.Ingress) error {
	_, err := i.VoyagerClient.Ingresses(i.Namespace()).Update(ing)
	return err
}

func (i *ingressInvocation) Delete(ing *api.Ingress) error {
	return i.VoyagerClient.Ingresses(i.Namespace()).Delete(ing.Name)
}

func (i *ingressInvocation) IsTargetCreated(ing *api.Ingress) bool {
	return i.Controller(ing).IsExists()
}

func (i *ingressInvocation) Controller(ing *api.Ingress) *ingress.Controller {
	return ingress.NewController(i.KubeClient, i.VoyagerClient, nil, i.VoyagerConfig(), ing)
}

func (i *ingressInvocation) EventuallyStarted(ing *api.Ingress) GomegaAsyncAssertion {
	return Eventually(func() bool {
		_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Get(ing.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false
		}

		_, err = i.KubeClient.CoreV1().Endpoints(i.Namespace()).Get(ing.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false
		}

		return true
	}, "10m", "20s")
}

func (i *ingressInvocation) GetHTTPEndpoints(ing *api.Ingress) ([]string, error) {
	switch ing.LBType() {
	case api.LBTypeLoadBalancer:
		return getLoadBalancerURLs(i.Config.CloudProviderName, i.KubeClient, ing)
	case api.LBTypeNodePort:
		return getHostPortURLs(i.Config.CloudProviderName, i.KubeClient, ing)
	}
	return nil, errors.New("LBType Not recognized")
}

func (i *ingressInvocation) GetOffShootService(ing *api.Ingress) (*v1.Service, error) {
	return i.KubeClient.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
}

func (i *ingressInvocation) DoHTTP(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *testserverclient.Response) bool) error {
	for _, url := range eps {
		resp, err := testserverclient.NewTestHTTPClient(url).Method(method).Path(path).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		log.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (i *ingressInvocation) DoHTTPWithHeader(retryCount int, ing *api.Ingress, eps []string, method, path string, h map[string]string, matcher func(resp *testserverclient.Response) bool) error {
	for _, url := range eps {
		resp, err := testserverclient.NewTestHTTPClient(url).Method(method).Header(h).Path(path).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		log.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (i *ingressInvocation) DoHTTPTestRedirect(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *testserverclient.Response) bool) error {
	for _, url := range eps {
		resp, err := testserverclient.NewTestHTTPClient(url).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
		if err != nil {
			return err
		}

		log.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (i *ingressInvocation) DoHTTPStatus(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *testserverclient.Response) bool) error {
	for _, url := range eps {
		resp, err := testserverclient.NewTestHTTPClient(url).Method(method).Path(path).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		log.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (i *ingressInvocation) DoTCP(retryCount int, ing *api.Ingress, eps []string, matcher func(resp *testserverclient.Response) bool) error {
	for _, url := range eps {
		resp, err := testserverclient.NewTestTCPClient(url).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		log.Infoln("TCP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}
