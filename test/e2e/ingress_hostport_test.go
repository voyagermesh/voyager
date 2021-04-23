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

// nolint:goconst
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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kmodules.xyz/client-go/meta"
)

var _ = Describe("IngressHostPort", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetDaemonSkeletonRule(ing)

		if !meta.PossiblyInCluster() && options.CloudProvider != api.ProviderMinikube {
			Skip("Test is Running from outside of cluster skipping test")
		}
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
			_ = f.Ingress.Delete(ing)
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Spec.Rules[0].HTTP.Port = intstr.FromInt(2001)
		})

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
			err := f.Ingress.Delete(ing)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return f.Ingress.IsExists(ing) == nil
			}, "5m", "10s").Should(BeFalse())
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			ing.Spec.Rules[0].HTTP.Port = intstr.FromInt(2002)
		})

		It("Should update Loadbalancer", func() {
			By("Updating Ingress resource")
			uing, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules[0].HTTP.Paths[0].Path = "/newtestpath"
			err = f.Ingress.Update(uing)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting some time for update to be applied")
			time.Sleep(time.Second * 10)

			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling new HTTP path")
			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/newtestpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/newtestpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking old path")
			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())
		})

		It("Should add TCP rule", func() {
			By("Updating Ingress resource for TCP")
			uing, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromString("2003"),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromString("4545"),
							},
						},
					},
				},
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromString("2004"),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromString("4545"),
							},
						},
					},
				},
			}

			err = f.Ingress.Update(uing)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				svc, err := f.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
				if err != nil {
					return err
				}
				for _, port := range svc.Spec.Ports {
					if port.Port == 2003 {
						return nil
					}
				}
				return errors.New("TCP port not found")
			}, "5m", "20s").Should(BeNil())

			if options.CloudProvider != api.ProviderMinikube {
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				By("Calling new TCP")
				err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *client.Response) bool {
					return Expect(r.ServerPort).Should(Equal(":4545"))
				})
				Expect(err).NotTo(HaveOccurred())

				err = f.Ingress.CheckTestServersPortAssignments(ing)
				Expect(err).Should(BeNil())
			}
		})
	})

	Describe("Frontend bind address (http)", func() {
		BeforeEach(func() {
			ing.Spec.Rules[0].HTTP.Port = intstr.FromInt(2001)
			ing.Spec.Rules[0].HTTP.Address = f.MinikubeIP()
		})

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

	Describe("Frontend bind address (tcp)", func() {
		BeforeEach(func() {
			svcName := ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port:    intstr.FromInt(4001),
							Address: f.MinikubeIP(),
							Backend: api.IngressBackend{
								ServiceName: svcName,
								ServicePort: intstr.FromInt(4545),
							},
						},
					},
				},
			}
		})

		It("Should response TCP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoTCP(framework.NoRetry, ing, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
