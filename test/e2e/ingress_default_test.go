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
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extensions "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressCoreOperations", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
		ext *extensions.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ext = &extensions.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: f.Namespace(),
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
											ServiceName: f.Ingress.TestServerName(),
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

		// This ensures compatibility with extensions.Ingress
		var err error
		ing, err = api.NewEngressFromIngress(ext)
		Expect(err).NotTo(HaveOccurred())

	})

	JustBeforeEach(func() {
		_, err := f.KubeClient.NetworkingV1beta1().Ingresses(ext.Namespace).Create(context.TODO(), ext, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if options.Cleanup {
			_ = f.KubeClient.NetworkingV1beta1().Ingresses(ext.Namespace).Delete(context.TODO(), ext.Name, metav1.DeleteOptions{})
		}
	})

	Describe("Create", func() {
		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Delete", func() {
		It("Should delete Ingress resource", func() {
			By("Deleting Ingress resource")
			err := f.KubeClient.NetworkingV1beta1().Ingresses(ext.Namespace).Delete(context.TODO(), ext.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return f.Ingress.IsExists(ing) == nil
			}, "5m", "10s").Should(BeFalse())
		})
	})

	Describe("Update", func() {
		It("Should update Loadbalancer", func() {
			By("Updating Ingress resource")
			uing, err := f.KubeClient.NetworkingV1beta1().Ingresses(ext.Namespace).Get(context.TODO(), ext.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestPath"
			_, err = f.KubeClient.NetworkingV1beta1().Ingresses(ext.Namespace).Update(context.TODO(), uing, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting some time for update to be applied")
			time.Sleep(time.Second * 10)

			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling new HTTP path")
			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/newTestPath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/newTestPath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking old path")
			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
