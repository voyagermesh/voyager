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
	core_util "kmodules.xyz/client-go/core/v1"
)

var _ = Describe("Ingress SSL Passthrough", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)

		// remove "ssl-verify-none" annotation from test-service
		meta := metav1.ObjectMeta{
			Name:      f.Ingress.TestServerHTTPSName(),
			Namespace: f.Namespace(),
		}
		_, _, err := core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(obj *core.Service) *core.Service {
			delete(obj.Annotations, "ingress.appscode.com/backend-tls")
			return obj
		}, metav1.PatchOptions{})
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
		// restore "ssl-verify-none" annotation in test-service
		meta := metav1.ObjectMeta{
			Name:      f.Ingress.TestServerHTTPSName(),
			Namespace: f.Namespace(),
		}
		_, _, err := core_util.CreateOrPatchService(context.TODO(), f.KubeClient, meta, func(obj *core.Service) *core.Service {
			obj.Annotations = map[string]string{
				"ingress.appscode.com/backend-tls": "ssl verify none",
			}
			return obj
		}, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		if options.Cleanup {
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
		}
	})

	Describe("With SSL Passthrough Annotation", func() {
		BeforeEach(func() {
			ing.Annotations[api.SSLPassthrough] = "true"
			ing.Annotations[api.SSLRedirect] = "false"
			ing.Spec = api.IngressSpec{
				Rules: []api.IngressRule{
					{
						Host: framework.TestDomain,
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Port: intstr.FromInt(8443),
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerHTTPSName(),
												ServicePort: intstr.FromInt(3443),
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("Should Open port 8443 in TCP mode", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(svc.Spec.Ports)).Should(Equal(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(8443)))

			err = f.Ingress.DoHTTPs(framework.MaxRetry, framework.TestDomain, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.Host).Should(Equal(framework.TestDomain))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
