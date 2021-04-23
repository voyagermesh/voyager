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

package e2e_test

import (
	"context"
	"net/http"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressWithErrorFiles", func() {
	var (
		f         *framework.Invocation
		ing       *api.Ingress
		configMap *core.ConfigMap
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		configMap = &core.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			Data: map[string]string{
				"503.http": `HTTP/1.0 400 Bad request
Cache-Control: no-cache
Connection: close
Content-Type: text/html

{"body": "haproxy-errorfile"}`,
			},
		}
		_, err := f.KubeClient.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		By("Creating ingress with name " + ing.GetName())
		err := f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if options.Cleanup {
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
			Expect(f.KubeClient.CoreV1().ConfigMaps(configMap.Namespace).Delete(context.TODO(), configMap.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		}
	})

	Describe("Ingress Error Files", func() {
		BeforeEach(func() {
			ing.Annotations[api.ErrorFiles] = configMap.Name
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/wrongpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusBadRequest)) &&
						Expect(r.Body).Should(Equal("haproxy-errorfile"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Ingress Error Files HostPort", func() {
		BeforeEach(func() {
			f.Ingress.SetDaemonSkeletonRule(ing)
			ing.Spec.Rules[0].HTTP.Port = intstr.FromInt(2001)
			ing.Annotations[api.ErrorFiles] = configMap.Name
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/wrongpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusBadRequest)) &&
						Expect(r.Body).Should(Equal("haproxy-errorfile"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Ingress Error Files NodePort", func() {
		BeforeEach(func() {
			ing.Annotations[api.LBType] = api.LBTypeNodePort
			ing.Spec.Rules[0].Host = framework.TestDomain
			ing.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(32368)
			ing.Annotations[api.ErrorFiles] = configMap.Name
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.NoRetry, framework.TestDomain, ing, eps, "GET", "/testpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.NoRetry, framework.TestDomain, ing, eps, "GET", "/wrongpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusBadRequest)) &&
						Expect(r.Body).Should(Equal("haproxy-errorfile"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Ingress Error Files (errorloc)", func() {
		BeforeEach(func() {
			configMap = &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				Data: map[string]string{
					"503": "errorloc http://echo.jsontest.com/status/200/body/haproxy-errorloc",
				},
			}
			_, err := f.KubeClient.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ing.Annotations[api.ErrorFiles] = configMap.Name
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			// Should redirect to echo.jsontest.com
			err = f.Ingress.DoHTTPTestRedirect(framework.NoRetry, ing, eps, "GET", "/wrongpath",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(302)) &&
						Expect(r.ResponseHeader.Get("Location")).Should(Equal("http://echo.jsontest.com/status/200/body/haproxy-errorloc"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
