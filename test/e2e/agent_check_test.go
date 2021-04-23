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
	"strings"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	"github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("With Agent Check", func() {
	var (
		f             *framework.Invocation
		ing           *api.Ingress
		meta          metav1.ObjectMeta
		svcAnnotation map[string]string
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	JustBeforeEach(func() {

		var err error
		meta, err = f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
		Expect(err).NotTo(HaveOccurred())

		ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.IngressBackend.ServiceName = meta.Name

		By("Creating ingress with name " + ing.GetName())
		err = f.Ingress.Create(ing)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if options.Cleanup {
			_ = f.Ingress.Delete(ing)
		}
	})

	Describe("With Correct Port and Default Agent Inter", func() {
		BeforeEach(func() {
			svcAnnotation = map[string]string{
				api.AgentPort: "5555",
			}
		})

		It("Should Response HTTP", func() {
			By("Getting Backend Service URL for Agent Check Port")
			svcURL, err := f.Ingress.GetNodePortServiceURLForSpecificPort(meta.Name, 5555)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svcURL)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, []string{svcURL}, func(r *client.Response) bool {
				return Expect(r.Body).Should(HavePrefix("up"))
			})
			Expect(err).NotTo(HaveOccurred())

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(5, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Correct Port and Default Agent Inter and Cookie Enabled", func() {
		BeforeEach(func() {
			svcAnnotation = map[string]string{
				api.AgentPort: "5555",
			}
			ing.Annotations[api.IngressAffinity] = "cookie"
		})

		It("Should Response HTTP", func() {
			By("Getting HTTP Endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Getting Backend Service URL for Agent Check Port")
			svcURL, err := f.Ingress.GetNodePortServiceURLForSpecificPort(meta.Name, 5555)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svcURL)).Should(BeNumerically(">=", 1))

			var lastCookie *http.Cookie

			err = f.Ingress.DoHTTPStatus(5, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				for _, cookie := range r.Cookies {
					if cookie.Name == ing.StickySessionCookieName() {
						lastCookie = cookie
						break
					}
				}
				return Expect(r.Status).Should(Equal(http.StatusOK))
			})
			Expect(err).NotTo(HaveOccurred())

			// apache bench until agent server responses "drain"

			for req := 1; ; req++ {

				// check agent response
				var response *client.Response

				err = f.Ingress.DoTCP(framework.MaxRetry, ing, []string{svcURL}, func(r *client.Response) bool {
					response = r
					return Expect(len(r.Body)).Should(BeNumerically(">=", 1))
				})
				Expect(err).NotTo(HaveOccurred())

				// if body == drain, retrieve the cookie
				// from last response and break the loop
				if strings.HasPrefix(response.Body, "drain") {
					break
				}

				if strings.HasPrefix(response.Body, "up") {
					_ = sh.Command("ab", "-k", "-n", "100000", "-c", "1000", eps[0]+"/testpath").Run()
				}
			}

			// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#5.2-agent-check
			// The word "drain". This will turn the server's administrative state to the
			// DRAIN mode, thus it will not accept any new connections other than those
			// that are accepted via persistence.
			// So, traffic using cooking persistence will be OK

			By("Requesting Without Cookie Should Not Respond")
			err = f.Ingress.DoHTTPStatus(5, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Requesting With Cookie Should Be OK")
			err = f.Ingress.DoHTTPStatusWithCookies(5, ing, eps, "GET", "/testpath/ok", []*http.Cookie{lastCookie}, func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Wrong Port and Default Agent Inter", func() {

		BeforeEach(func() {
			svcAnnotation = map[string]string{
				api.AgentPort: "5553",
			}
		})

		It("Should Response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#5.2-agent-check
			// Failure to connect to the agent is not considered an error as connectivity
			// is tested by the regular health check which is enabled by the "check"
			// parameter.

			err = f.Ingress.DoHTTPStatus(5, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Correct Port and Customized Agent Inter", func() {
		BeforeEach(func() {
			svcAnnotation = map[string]string{
				api.AgentPort:     "5555",
				api.AgentInterval: "1s",
			}
		})

		It("Should Response HTTP OK", func() {
			By("Getting Backend Service URL for Agent Check Port")
			svcURL, err := f.Ingress.GetNodePortServiceURLForSpecificPort(meta.Name, 5555)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svcURL)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoTCP(framework.MaxRetry, ing, []string{svcURL}, func(r *client.Response) bool {
				return Expect(r.Body).Should(HavePrefix("up"))
			})
			Expect(err).NotTo(HaveOccurred())

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Checking /testpath/ok for " + eps[0])
			err = f.Ingress.DoHTTPStatus(framework.MaxRetry, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
