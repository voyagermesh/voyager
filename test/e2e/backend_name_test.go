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
	"net/http"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Haproxy backend names", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
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
			err := f.Ingress.Delete(ing)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("Multiple bind addresses with identical host, port, path and backend svc", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Address: "0.0.0.0",
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: f.Ingress.TestServerName(),
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Address: "::",
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: f.Ingress.TestServerName(),
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
			}
		})

		Context("With auto-generated backend names", func() {
			It("Should response http", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With specified backend names", func() {
			BeforeEach(func() {
				ing.Spec.Rules[0].HTTP.Paths[0].Backend.Name = "be-ipv4"
				ing.Spec.Rules[1].HTTP.Paths[0].Backend.Name = "be-ipv6"
			})
			It("Should response http", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
