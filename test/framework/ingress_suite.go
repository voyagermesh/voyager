/*
Copyright AppsCode Inc. and Contributors

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
	"context"
	"crypto/tls"
	"net/http"
	"strconv"
	"strings"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

const (
	testServerImage = "appscode/test-server:2.4"
)

var (
	testServerResourceName      = "e2e-test-server-" + rand.Characters(5)
	testServerHTTPSResourceName = "e2e-test-server-https-" + rand.Characters(5)
	emptyServiceName            = "e2e-empty-" + rand.Characters(5)
)

func (ni *ingressInvocation) Setup() error {
	if err := ni.setupTestServers(); err != nil {
		return err
	}
	return ni.waitForTestServer()
}

func (ni *ingressInvocation) Teardown() {
	if ni.Cleanup {
		Expect(ni.KubeClient.CoreV1().Services(ni.Namespace()).Delete(context.TODO(), testServerResourceName, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		Expect(ni.KubeClient.CoreV1().Services(ni.Namespace()).Delete(context.TODO(), testServerHTTPSResourceName, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		Expect(ni.KubeClient.CoreV1().Services(ni.Namespace()).Delete(context.TODO(), emptyServiceName, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		_, err := ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Get(context.TODO(), testServerResourceName, metav1.GetOptions{})
		if err == nil {
			Expect(ni.KubeClient.AppsV1().Deployments(ni.Namespace()).Delete(context.TODO(), testServerResourceName, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		}
		list, err := ni.VoyagerClient.VoyagerV1beta1().Ingresses(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			for _, ing := range list.Items {
				Expect(ni.VoyagerClient.VoyagerV1beta1().Ingresses(ing.Namespace).Delete(context.TODO(), ing.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			}
		}
	}
}

func (ni *ingressInvocation) TestServerName() string {
	return testServerResourceName
}

func (ni *ingressInvocation) EmptyServiceName() string {
	return emptyServiceName
}

func (ni *ingressInvocation) TestServerHTTPSName() string {
	return testServerHTTPSResourceName
}

func (ni *ingressInvocation) Create(ing *api.Ingress) error {
	_, err := ni.VoyagerClient.VoyagerV1beta1().Ingresses(ni.Namespace()).Create(context.TODO(), ing, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	go ni.printInfoForDebug(ing)
	return nil
}

func (ni *ingressInvocation) printInfoForDebug(ing *api.Ingress) {
	for {
		pods, err := ni.KubeClient.CoreV1().Pods(ing.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set(ing.OffshootSelector())).String(),
		})
		if err == nil {
			if len(pods.Items) > 0 {
				for _, pod := range pods.Items {
					klog.Warningln("Log: $ kubectl logs -f", pod.Name, "-n", ing.Namespace)
					klog.Warningln("Exec: $ kubectl exec", pod.Name, "-n", ing.Namespace, "sh")
				}
				return
			}
		}
		time.Sleep(time.Second * 2)
	}
}

func (ni *ingressInvocation) Get(ing *api.Ingress) (*api.Ingress, error) {
	return ni.VoyagerClient.VoyagerV1beta1().Ingresses(ni.Namespace()).Get(context.TODO(), ing.Name, metav1.GetOptions{})
}

func (ni *ingressInvocation) Update(ing *api.Ingress) error {
	_, err := ni.VoyagerClient.VoyagerV1beta1().Ingresses(ni.Namespace()).Update(context.TODO(), ing, metav1.UpdateOptions{})
	return err
}

func (ni *ingressInvocation) Delete(ing *api.Ingress) error {
	return ni.VoyagerClient.VoyagerV1beta1().Ingresses(ni.Namespace()).Delete(context.TODO(), ing.Name, metav1.DeleteOptions{})
}

func (ni *ingressInvocation) IsExistsEventually(ing *api.Ingress) bool {
	return Eventually(func() error {
		err := ni.IsExists(ing)
		if err != nil {
			klog.Errorln("IsExistsEventually failed with error,", err)
		}
		return err
	}, "5m", "10s").Should(BeNil())
}

func (ni *ingressInvocation) IsExists(ing *api.Ingress) error {
	var err error
	_, err = ni.KubeClient.AppsV1().Deployments(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}

	_, err = ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}

	_, err = ni.KubeClient.CoreV1().ConfigMaps(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (ni *ingressInvocation) EventuallyStarted(ing *api.Ingress) GomegaAsyncAssertion {
	return Eventually(func() bool {
		_, err := ni.KubeClient.CoreV1().Services(ni.Namespace()).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
		if err != nil {
			return false
		}

		if ing.LBType() != api.LBTypeHostPort {
			_, err = ni.KubeClient.CoreV1().Endpoints(ni.Namespace()).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
			if err != nil {
				return false
			}
		}
		return true
	}, "10m", "20s")
}

func (ni *ingressInvocation) GetHTTPEndpoints(ing *api.Ingress) ([]string, error) {
	switch ing.LBType() {
	case api.LBTypeLoadBalancer:
		return ni.getLoadBalancerURLs(ing)
	case api.LBTypeHostPort:
		return ni.getHostPortURLs(ing)
	case api.LBTypeNodePort:
		return ni.getNodePortURLs(ing)
	}
	return nil, errors.New("LBType Not recognized")
}

func (ni *ingressInvocation) FilterEndpointsForPort(eps []string, port core.ServicePort) []string {
	ret := make([]string, 0)
	for _, p := range eps {
		if strings.HasSuffix(p, ":"+strconv.FormatInt(int64(port.Port), 10)) ||
			strings.HasSuffix(p, ":"+strconv.FormatInt(int64(port.NodePort), 10)) {
			ret = append(ret, p)
		}
	}
	return ret
}

func (ni *ingressInvocation) GetOffShootService(ing *api.Ingress) (*core.Service, error) {
	return ni.KubeClient.CoreV1().Services(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
}

func (ni *ingressInvocation) GetFreeNodePort(p int32) int {
	svc, err := ni.KubeClient.CoreV1().Services(core.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
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

func (ni *ingressInvocation) DoHTTP(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPWithTimeout(retryCount int, timeout int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClientWithTimeout(url, timeout).WithHost(host).Method(method).Path(path).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPWithHeader(retryCount int, ing *api.Ingress, eps []string, method, path string, h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Header(h).Path(path).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPs(retryCount int, host, cert string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
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

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPsWithTransport(retryCount int, host string, tr *http.Transport, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}

		cl := client.NewTestHTTPClient(url).WithHost(host).WithTransport(tr).Method(method).Path(path)
		resp, err := cl.DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPTestRedirect(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPTestRedirectWithHost(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPTestRedirectWithHeader(retryCount int, host string, ing *api.Ingress, eps []string, method, path string,
	h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Header(h).Path(path).DoTestRedirectWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPStatus(retryCount int, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPStatusWithCookies(retryCount int, ing *api.Ingress, eps []string, method, path string, cookies []*http.Cookie, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Path(path).Cookie(cookies).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPStatusWithHost(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPsStatus(retryCount int, host string, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}
		resp, err := client.NewTestHTTPClient(url).WithHost(host).Method(method).Path(path).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoTestRedirectWithTransport(retryCount int, host string, tr *http.Transport, ing *api.Ingress, eps []string, method, path string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		if strings.HasPrefix(url, "http://") {
			url = "https://" + url[len("http://"):]
		}
		resp, err := client.NewTestHTTPClient(url).WithTransport(tr).WithHost(host).Method(method).Path(path).DoTestRedirectWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPStatusWithHeader(retryCount int, ing *api.Ingress, eps []string, method, path string, h map[string]string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestHTTPClient(url).Method(method).Header(h).Path(path).DoStatusWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoHTTPWithSNI(retryCount int, host string, eps []string, matcher func(resp *client.Response) bool) error {
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

		klog.Infoln("HTTP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoTCP(retryCount int, ing *api.Ingress, eps []string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestTCPClient(url).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("TCP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}

func (ni *ingressInvocation) DoTCPWithSSL(retryCount int, cert string, ing *api.Ingress, eps []string, matcher func(resp *client.Response) bool) error {
	for _, url := range eps {
		resp, err := client.NewTestTCPClient(url).WithSSL(cert).DoWithRetry(retryCount)
		if err != nil {
			return err
		}

		klog.Infoln("TCP Response received from server", *resp)
		if !matcher(resp) {
			return errors.New("Failed to match")
		}
	}
	return nil
}
