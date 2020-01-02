/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/test-server/client"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testServerImage = "appscode/test-server:2.4"
	MinikubeIP      = "192.168.99.100"
)

var (
	testServerResourceName      = "e2e-test-server-" + rand.Characters(5)
	testServerHTTPSResourceName = "e2e-test-server-https-" + rand.Characters(5)
	emptyServiceName            = "e2e-empty-" + rand.Characters(5)
)

func (i *ingressInvocation) Setup() error {
	if err := i.setupTestServers(); err != nil {
		return err
	}
	return i.waitForTestServer()
}

func (i *ingressInvocation) Teardown() {
	if i.Cleanup {
		Expect(i.KubeClient.CoreV1().Services(i.Namespace()).Delete(testServerResourceName, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
		Expect(i.KubeClient.CoreV1().Services(i.Namespace()).Delete(testServerHTTPSResourceName, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
		Expect(i.KubeClient.CoreV1().Services(i.Namespace()).Delete(emptyServiceName, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
		_, err := i.KubeClient.AppsV1().Deployments(i.Namespace()).Get(testServerResourceName, metav1.GetOptions{})
		if err == nil {
			Expect(i.KubeClient.AppsV1().Deployments(i.Namespace()).Delete(testServerResourceName, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
		}
		list, err := i.VoyagerClient.VoyagerV1beta1().Ingresses(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err == nil {
			for _, ing := range list.Items {
				Expect(i.VoyagerClient.VoyagerV1beta1().Ingresses(ing.Namespace).Delete(ing.Name, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
			}
		}
	}
}

func (i *ingressInvocation) TestServerName() string {
	return testServerResourceName
}

func (i *ingressInvocation) EmptyServiceName() string {
	return emptyServiceName
}

func (i *ingressInvocation) TestServerHTTPSName() string {
	return testServerHTTPSResourceName
}

func (i *ingressInvocation) Create(ing *api.Ingress) error {
	_, err := i.VoyagerClient.VoyagerV1beta1().Ingresses(i.Namespace()).Create(ing)
	if err != nil {
		return err
	}
	go i.printInfoForDebug(ing)
	return nil
}

func (i *ingressInvocation) printInfoForDebug(ing *api.Ingress) {
	for {
		pods, err := i.KubeClient.CoreV1().Pods(ing.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set(ing.OffshootSelector())).String(),
		})
		if err == nil {
			if len(pods.Items) > 0 {
				for _, pod := range pods.Items {
					log.Warningln("Log: $ kubectl logs -f", pod.Name, "-n", ing.Namespace)
					log.Warningln("Exec: $ kubectl exec", pod.Name, "-n", ing.Namespace, "sh")
				}
				return
			}
		}
		time.Sleep(time.Second * 2)
	}
}

func (i *ingressInvocation) Get(ing *api.Ingress) (*api.Ingress, error) {
	return i.VoyagerClient.VoyagerV1beta1().Ingresses(i.Namespace()).Get(ing.Name, metav1.GetOptions{})
}

func (i *ingressInvocation) Update(ing *api.Ingress) error {
	_, err := i.VoyagerClient.VoyagerV1beta1().Ingresses(i.Namespace()).Update(ing)
	return err
}

func (i *ingressInvocation) Delete(ing *api.Ingress) error {
	return i.VoyagerClient.VoyagerV1beta1().Ingresses(i.Namespace()).Delete(ing.Name, &metav1.DeleteOptions{})
}

func (i *ingressInvocation) IsExistsEventually(ing *api.Ingress) bool {
	return Eventually(func() error {
		err := i.IsExists(ing)
		if err != nil {
			log.Errorln("IsExistsEventually failed with error,", err)
		}
		return err
	}, "5m", "10s").Should(BeNil())
}

func (i *ingressInvocation) IsExists(ing *api.Ingress) error {
	var err error
	_, err = i.KubeClient.AppsV1().Deployments(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}

	_, err = i.KubeClient.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}

	_, err = i.KubeClient.CoreV1().ConfigMaps(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (i *ingressInvocation) EventuallyStarted(ing *api.Ingress) GomegaAsyncAssertion {
	return Eventually(func() bool {
		_, err := i.KubeClient.CoreV1().Services(i.Namespace()).Get(ing.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false
		}

		if ing.LBType() != api.LBTypeHostPort {
			_, err = i.KubeClient.CoreV1().Endpoints(i.Namespace()).Get(ing.OffshootName(), metav1.GetOptions{})
			if err != nil {
				return false
			}
		}
		return true
	}, "10m", "20s")
}

func (i *ingressInvocation) GetHTTPEndpoints(ing *api.Ingress) ([]string, error) {
	switch ing.LBType() {
	case api.LBTypeLoadBalancer:
		return i.getLoadBalancerURLs(ing)
	case api.LBTypeHostPort:
		return i.getHostPortURLs(ing)
	case api.LBTypeNodePort:
		return i.getNodePortURLs(ing)
	}
	return nil, errors.New("LBType Not recognized")
}

func (i *ingressInvocation) FilterEndpointsForPort(eps []string, port core.ServicePort) []string {
	ret := make([]string, 0)
	for _, p := range eps {
		if strings.HasSuffix(p, ":"+strconv.FormatInt(int64(port.Port), 10)) ||
			strings.HasSuffix(p, ":"+strconv.FormatInt(int64(port.NodePort), 10)) {
			ret = append(ret, p)
		}
	}
	return ret
}

func (i *ingressInvocation) GetOffShootService(ing *api.Ingress) (*core.Service, error) {
	return i.KubeClient.CoreV1().Services(ing.Namespace).Get(ing.OffshootName(), metav1.GetOptions{})
}

func (i *ingressInvocation) GetFreeNodePort(p int32) int {
	svc, err := i.KubeClient.CoreV1().Services(core.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return int(p)
	}
	return int(getFreeNodePort(svc.Items, p))
}

func getFreeNodePort(svc []core.Service, p int32) int32 {
	for _, service := range svc {
		for _, ports := range service.Spec.Ports {
			if ports.NodePort == p {
				// Port already in use. Try for the next port
				p++
				return getFreeNodePort(svc, p)
			}
		}
	}
	return p
}

func (i *ingressInvocation) DoHTTP(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPWithTimeout(retryCount int, timeout int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClientWithTimeout(url, timeout).WithHost(host).Method(method).Path(path).DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPWithHeader(retryCount int, ing *api.Ingress, eps []string, method, path string, h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Header(h).Path(path).DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPs(retryCount int, host, cert string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}

		cl := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path)
		if len(cert) > 0 {
			cl = cl.WithCert(cert)
		} else {
			cl = cl.WithInsecure()
		}

		resp, err := cl.DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPsWithTransport(retryCount int, host string, tr *http.Transport, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}

		cl := client.NewTestHTTPClient(url).WithHost(host).WithTransport(tr).Method(method).Path(path)
		resp, err := cl.DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPTestRedirect(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPTestRedirectWithHost(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPTestRedirectWithHeader(retryCount int, host string, ing *api.Ingress, eps []string, method, path string,
	h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Header(h).Path(path).DoTestRedirectWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPStatus(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).DoStatusWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPStatusWithCookies(retryCount int, ing *api.Ingress, eps []string, method, path string, cookies []*http.Cookie, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).Cookie(cookies).DoStatusWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPStatusWithHost(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoStatusWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPsStatus(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoStatusWithRetry(retryCount)
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

func (i *ingressInvocation) DoTestRedirectWithTransport(retryCount int, host string, tr *http.Transport, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}
		resp, err := client.NewTestHTTPClient(url).WithTransport(tr).WithHost(host).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPStatusWithHeader(retryCount int, ing *api.Ingress, eps []string, method, path string, h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Header(h).Path(path).DoStatusWithRetry(retryCount)
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

func (i *ingressInvocation) DoHTTPWithSNI(retryCount int, host string, eps []string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
			},
		}

		resp, err := client.NewTestHTTPClient(url).WithHost(host).WithTransport(tr).DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoTCP(retryCount int, ing *api.Ingress, eps []string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestTCPClient(url).DoWithRetry(retryCount)
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

func (i *ingressInvocation) DoTCPWithSSL(retryCount int, cert string, ing *api.Ingress, eps []string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestTCPClient(url).WithSSL(cert).DoWithRetry(retryCount)
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
