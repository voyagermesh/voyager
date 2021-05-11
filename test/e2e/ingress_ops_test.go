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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

var _ = Describe("IngressOperations", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
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

	var (
		shouldResponseHTTP = func() {
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
		}

		shouldDeleteResource = func() {
			By("Deleting Ingress resource")
			err := f.Ingress.Delete(ing)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return f.Ingress.IsExists(ing) == nil
			}, "5m", "10s").Should(BeFalse())
		}
	)

	Describe("Create", func() {
		It("Should create Loadbalancer entry", func() {
			By("Checking StatusIP for provider: " + options.CloudProvider)
			if options.CloudProvider == api.ProviderMinikube {
				Skip("Minikube do not support this")
			}

			svc, err := f.Ingress.GetServiceWithLoadBalancerStatus(ing.OffshootName(), ing.Namespace)
			Expect(err).NotTo(HaveOccurred())

			baseIngress, err := f.Ingress.GetIngressWithLoadBalancerStatus(ing.Name, ing.Namespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(baseIngress.Status.LoadBalancer.Ingress)).Should(Equal(len(svc.Status.LoadBalancer.Ingress)))
			Expect(baseIngress.Status.LoadBalancer.Ingress[0]).Should(Equal(svc.Status.LoadBalancer.Ingress[0]))
		})

		It("Should response HTTP", shouldResponseHTTP)

		Describe("With persistent IP", func() {
			BeforeEach(func() {
				ing.Annotations[api.LoadBalancerIP] = options.LBPersistIP
			})
			It("Should persist service IP", func() {
				if len(options.LBPersistIP) == 0 {
					Skip("Persistent IP is not provided")
				}
				if !f.Ingress.SupportsServiceIP() {
					Skip("Persistent IP is not supported")
				}

				By("Checking HTTP Response", shouldResponseHTTP)
				oldsvc, err := f.Ingress.GetOffShootService(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(oldsvc.Status.LoadBalancer.Ingress)).Should(Equal(1))

				Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
				By("Wait for resource to be deleted", shouldDeleteResource)
				ing.Annotations[api.LoadBalancerIP] = options.LBPersistIP
				Expect(f.Ingress.Create(ing)).NotTo(HaveOccurred())

				By("Checking HTTP Response", shouldResponseHTTP)
				newsvc, err := f.Ingress.GetOffShootService(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(newsvc.Status.LoadBalancer.Ingress)).Should(Equal(1))

				Expect(newsvc.Status.LoadBalancer.Ingress).Should(Equal(oldsvc.Status.LoadBalancer.Ingress))
			})
		})

		Describe("With NodePort Service", func() {
			BeforeEach(func() {
				ing.Annotations[api.LBType] = api.LBTypeNodePort
				ing.Spec.Rules[0].HTTP.NodePort = intstr.FromInt(32345)
			})
			It("Should create nodeport service", func() {
				var svc *core.Service
				Eventually(func() error {
					var err error
					svc, err = f.Ingress.GetOffShootService(ing)
					return err
				}, "10m", "5s").Should(BeNil())
				Expect(svc).ShouldNot(BeNil())
				Expect(svc.Spec.Type).Should(Equal(core.ServiceTypeNodePort))
				Expect(svc.Spec.Ports[0].NodePort).Should(Equal(int32(32345)))
			})
		})

		Describe("With custom target annotations", func() {
			BeforeEach(func() {
				ing.Annotations[api.ServiceAnnotations] = `{"foo": "bar", "service-annotation": "set"}`
				ing.Annotations[api.PodAnnotations] = `{"foo": "bar", "pod-annotation": "set"}`
			})
			It("Should persist service IP", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				svc, err := f.Ingress.GetOffShootService(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(svc.Annotations).NotTo(BeNil())
				Expect(svc.Annotations).Should(HaveKey("foo"))
				Expect(svc.Annotations["foo"]).Should(Equal("bar"))
				Expect(svc.Annotations).Should(HaveKey("service-annotation"))
				Expect(svc.Annotations["service-annotation"]).Should(Equal("set"))

				pods, err := f.Ingress.KubeClient.CoreV1().Pods(svc.Namespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pods.Items)).Should(BeNumerically(">=", 1))
				for _, pod := range pods.Items {
					Expect(pod.Annotations).NotTo(BeNil())
					Expect(pod.Annotations).Should(HaveKey("foo"))
					Expect(pod.Annotations["foo"]).Should(Equal("bar"))
					Expect(pod.Annotations).Should(HaveKey("pod-annotation"))
					Expect(pod.Annotations["pod-annotation"]).Should(Equal("set"))
				}

				toBeUpdated, err := f.Ingress.Get(ing)
				Expect(err).NotTo(HaveOccurred())
				toBeUpdated.Annotations[api.ServiceAnnotations] = `{"bar": "foo", "second-service-annotation": "set"}`
				err = f.Ingress.Update(toBeUpdated)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(time.Second * 10)
				Eventually(func() bool {
					svc, err := f.Ingress.GetOffShootService(ing)
					return err == nil &&
						svc.Annotations["bar"] == "foo" &&
						svc.Annotations["second-service-annotation"] == "set"
				}, "5m", "5s").Should(BeTrue())

				toBeUpdated, err = f.Ingress.Get(ing)
				Expect(err).NotTo(HaveOccurred())
				toBeUpdated.Annotations[api.PodAnnotations] = `{"bar": "foo", "second-pod-annotation": "set"}`
				err = f.Ingress.Update(toBeUpdated)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(time.Second * 10)
				Eventually(func() bool {
					pods, err = f.Ingress.KubeClient.CoreV1().Pods(svc.Namespace).List(context.TODO(), metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
					})
					return err == nil &&
						len(pods.Items) >= 1 &&
						pods.Items[0].Annotations["bar"] == "foo" &&
						pods.Items[0].Annotations["second-pod-annotation"] == "set"
				}, "5m", "5s").Should(BeTrue())
			})
		})

		Describe("With Stats", func() {
			BeforeEach(func() {
				ing.Annotations[api.StatsOn] = `true`
				ing.Annotations[api.StatsPort] = `8787`
			})
			It("Should test stat service", func() {
				var svc *core.Service
				Eventually(func() error {
					var err error
					svc, err = f.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(context.TODO(), ing.StatsServiceName(), metav1.GetOptions{})
					return err
				}, "10m", "5s").Should(BeNil())
				Expect(svc).ShouldNot(BeNil())
				Expect(len(svc.Spec.Ports)).Should(Equal(1))
				Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(8787)))

				tobeUpdated, err := f.Ingress.Get(ing)
				Expect(err).NotTo(HaveOccurred())

				delete(tobeUpdated.Annotations, api.StatsOn)
				Expect(tobeUpdated.Annotations).ShouldNot(HaveKey(api.StatsOn))

				err = f.Ingress.Update(tobeUpdated)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() bool {
					_, err := f.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(context.TODO(), ing.StatsServiceName(), metav1.GetOptions{})
					return err != nil
				}, "2m", "20s").Should(BeTrue())
			})
		})

		Describe("Keep LoadBalancerSourceRanges for Service", func() {
			BeforeEach(func() {
				ing.Spec.LoadBalancerSourceRanges = []string{
					"192.101.0.0/16",
					"192.0.0.0/24",
				}
			})
			It("Should keep LoadBalancerSourceRanges", func() {
				var svc *core.Service
				Eventually(func() error {
					var err error
					svc, err = f.Ingress.GetOffShootService(ing)
					return err
				}, "10m", "5s").Should(BeNil())
				Expect(svc).ShouldNot(BeNil())
				Expect(len(svc.Spec.LoadBalancerSourceRanges)).Should(Equal(len(ing.Spec.LoadBalancerSourceRanges)))
				Expect(svc.Spec.LoadBalancerSourceRanges).Should(Equal(ing.Spec.LoadBalancerSourceRanges))

				tobeUpdated, err := f.Ingress.Get(ing)
				Expect(err).NotTo(HaveOccurred())
				tobeUpdated.Spec.LoadBalancerSourceRanges = []string{"192.10.0.0/24"}
				err = f.Ingress.Update(tobeUpdated)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() int {
					var err error
					svc, err = f.Ingress.GetOffShootService(ing)
					Expect(err).NotTo(HaveOccurred())
					return len(svc.Spec.LoadBalancerSourceRanges)
				}, "10m", "5s").Should(Equal(1))
				Expect(len(svc.Spec.LoadBalancerSourceRanges)).Should(Equal(len(tobeUpdated.Spec.LoadBalancerSourceRanges)))
				Expect(svc.Spec.LoadBalancerSourceRanges).Should(Equal(tobeUpdated.Spec.LoadBalancerSourceRanges))
			})
		})

		Describe("With Rules", func() {
			BeforeEach(func() {
				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
											},
											HeaderRules: []string{
												"X-Ingress-Test-Header ingress.appscode.com",
											},
											RewriteRules: []string{
												`^([^\ :]*)\ /(.*)$ \1\ /override/\2`,
											},
										},
									},
								},
							},
						},
					},
				}
			})

			It("Rules Should Apply", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/override/testpath/ok")) &&
							Expect(r.RequestHeaders.Get("X-Ingress-Test-Header")).Should(Equal("ingress.appscode.com"))
					})
				Expect(err).NotTo(HaveOccurred())

				err = f.Ingress.DoHTTPWithHeader(framework.MaxRetry,
					ing,
					eps,
					"GET",
					"/testpath/ok",
					map[string]string{
						"X-Ingress-Test-Header": api.SchemeGroupVersion.Group + "/v1beta1",
					},
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/override/testpath/ok")) &&
							Expect(r.RequestHeaders.Get("X-Ingress-Test-Header")).Should(Equal("ingress.appscode.com"))
					},
				)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With BackendRules", func() {
			BeforeEach(func() {
				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/old",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"acl add_url capture.req.uri -m beg /old/add/now",
													`http-response set-header X-Added-From-Proxy added-from-proxy if add_url`,

													"acl rep_url path_beg /old/replace",
													`reqrep ^([^\ :]*)\ /(.*)$ \1\ /rewrited/from/proxy/\2 if rep_url`,
												},
											},
										},
									},
									{
										Path: "/test-second",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: f.Ingress.TestServerName(),
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"acl add_url capture.req.uri -m beg /test-second",
													`http-response set-header X-Added-From-Proxy added-from-proxy if add_url`,

													"acl rep_url path_beg /test-second",
													`reqrep ^([^\ :]*)\ /(.*)$ \1\ /rewrited/from/proxy/\2 if rep_url`,
												},
											},
											HeaderRules: []string{
												"X-Ingress-Test-Header ingress.appscode.com",
											},
											RewriteRules: []string{
												`^([^\ :]*)\ /(.*)$ \1\ /override/\2`,
											},
										},
									},
								},
							},
						},
					},
				}
			})
			It("BackendRules Should Apply", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/old/replace", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/rewrited/from/proxy/old/replace"))
				})
				Expect(err).NotTo(HaveOccurred())

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/old/add/now", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/old/add/now")) &&
						Expect(r.ResponseHeader.Get("X-Added-From-Proxy")).Should(Equal("added-from-proxy"))
				})
				Expect(err).NotTo(HaveOccurred())

				err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/test-second", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/override/rewrited/from/proxy/test-second")) &&
						Expect(r.ResponseHeader.Get("X-Added-From-Proxy")).Should(Equal("added-from-proxy")) &&
						Expect(r.RequestHeaders.Get("X-Ingress-Test-Header")).Should(Equal("ingress.appscode.com"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Delete", func() {
		It("Should delete Ingress resource", shouldDeleteResource)
	})

	Describe("Update", func() {
		It("Should update Loadbalancer", func() {
			By("Updating Ingress resource")
			uing, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestPath"
			err = f.Ingress.Update(uing)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting some time for update to be applied")
			time.Sleep(time.Second * 5)

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

		It("Should add TCP rule", func() {
			By("Updating Ingress resource for TCP")
			uing, err := f.Ingress.Get(ing)
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						TCP: &api.TCPIngressRuleValue{
							Port: intstr.FromString("4545"),
							Backend: api.IngressBackend{
								ServiceName: f.Ingress.TestServerName(),
								ServicePort: intstr.FromString("4545"),
							},
						},
					},
				},
				// Broken in Kube 1.8
				// ref: https://github.com/kubernetes/kubernetes/issues/47222
				//{
				//	IngressRuleValue: api.IngressRuleValue{
				//		TCP: &api.TCPIngressRuleValue{
				//			Port: intstr.FromString("4949"),
				//			Backend: api.IngressBackend{
				//				ServiceName: f.Ingress.TestServerName(),
				//				ServicePort: intstr.FromString("4545"),
				//			},
				//		},
				//	},
				//},
			}
			err = f.Ingress.Update(uing)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting some time for update to be applied")
			Eventually(func() error {
				svc, err := f.KubeClient.CoreV1().Services(ing.GetNamespace()).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
				if err != nil {
					return err
				}
				for _, port := range svc.Spec.Ports {
					if port.Port == 4545 {
						return nil
					}
				}
				return errors.New("TCP port not found")
			}, "5m", "10s").Should(BeNil())

			By("Getting endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling new TCP")
			err = f.Ingress.DoTCP(framework.MaxRetry, ing, eps, func(r *client.Response) bool {
				return Expect(r.ServerPort).Should(Equal(":4545"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking TestServers PortAssignments")
			err = f.Ingress.CheckTestServersPortAssignments(ing)
			Expect(err).Should(BeNil())
		})
	})

	Describe("With sticky session", func() {
		BeforeEach(func() {
			ing.Annotations[api.IngressAffinity] = "cookie"
			ing.Annotations[api.IngressAffinitySessionCookieName] = "TEST-COOKIE_NAME"
			ing.Annotations[api.IngressAffinitySessionCookieHash] = "md5"
		})

		It("Should Stick Session", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok")) &&
					Expect(r.ResponseHeader.Get("Set-Cookie")).ShouldNot(BeEmpty()) &&
					Expect(r.ResponseHeader.Get("Set-Cookie")).To(HavePrefix("TEST-COOKIE_NAME="))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With CORS Enabled", func() {
		shouldResponseCORS := func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok")) &&
						Expect(r.ResponseHeader.Get("Access-Control-Allow-Origin")).Should(Equal("*")) &&
						Expect(r.ResponseHeader.Get("Access-Control-Allow-Credentials")).Should(Equal("true")) &&
						Expect(r.ResponseHeader.Get("Access-Control-Allow-Methods")).Should(Equal(api.CORSDefaultAllowedMethods)) &&
						Expect(r.ResponseHeader.Get("Access-Control-Allow-Headers")).Should(Equal(api.CORSDefaultAllowedHeaders))
				})
			Expect(err).NotTo(HaveOccurred())
		}
		Context("Engress key", func() {
			BeforeEach(func() {
				ing.Annotations[api.CORSEnabled] = "true"
			})
			It("Should Response CORS", shouldResponseCORS)
		})
		Context("Origin and Method specified", func() {
			BeforeEach(func() {
				ing.Annotations[api.CORSEnabled] = "true"
				ing.Annotations[api.CORSAllowedOrigin] = "test.e2e"
				ing.Annotations[api.CORSAllowedMethods] = "GET,PUT"
			})
			It("Should Response CORS", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET", "/testpath/ok",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/testpath/ok")) &&
							Expect(r.ResponseHeader.Get("Access-Control-Allow-Origin")).Should(Equal("test.e2e")) &&
							Expect(r.ResponseHeader.Get("Access-Control-Allow-Credentials")).Should(Equal("true")) &&
							Expect(r.ResponseHeader.Get("Access-Control-Allow-Methods")).Should(Equal("GET,PUT")) &&
							Expect(r.ResponseHeader.Get("Access-Control-Allow-Headers")).Should(Equal(api.CORSDefaultAllowedHeaders))
					})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("With Body Size Enabled", func() {
		BeforeEach(func() {
			ing.Annotations[api.ProxyBodySize] = "500"
		})

		It("Should Response Deny", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatusWithHeader(framework.NoRetry, ing, eps, "GET", "/testpath/ok", map[string]string{
				"Content-Length": "600",
			}, func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusBadRequest))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Whitelist Specified", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
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
							Port: intstr.FromInt(3124),
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
			f.Ingress.SetSkeletonFrontendRule(ing)
			ing.Annotations[api.WhitelistSourceRange] = f.MinikubeIP()
		})

		It("Should Add Whitelisted Ips", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			// Manually check if whitelisted ips are added to each frontend rule of generated HAProxy config
			// TODO @ dipta: how to test if whitelist is actually working?
		})
	})

	Describe("With Global MaxConnections (1) Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.MaxConnections] = "1"
			ing.Annotations[api.DefaultsTimeOut] = `{"connect": "300s", "server": "300s"}`
		})

		It("Should Allow 1 Connection Concurrently", func() {
			By("Getting HTTP endpoints")

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			errChan := make(chan error)
			go func() {
				// request-1: take 30s to response
				errChan <- f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 300, "", ing, eps, "GET",
					"/testpath/ok?delay=30s",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/testpath/ok"))
					})
			}()

			time.Sleep(time.Second * 5) // to ensure request-1 always hits server before request-2

			// request-2: responses instantaneously
			err = f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 5, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})

			// request-1 should block request-2 since maxconn = 1
			// request-2 should be timeout (sleep: 5s + client-timeout: 5s < request-1: 30s)
			Expect(err).To(HaveOccurred())
			Expect(<-errChan).NotTo(HaveOccurred()) // check request-1

		})
	})

	Describe("With Global MaxConnections (2) Specified", func() {
		BeforeEach(func() {
			ing.Annotations[api.MaxConnections] = "2"
			ing.Annotations[api.DefaultsTimeOut] = `{"connect": "300s", "server": "300s"}`
		})

		It("Should Allow 2 Connections Concurrently", func() {
			By("Getting HTTP endpoints")

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			errChan := make(chan error)
			go func() {
				// request-1: take 30s to response
				errChan <- f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 300, "", ing, eps, "GET",
					"/testpath/ok?delay=30s",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/testpath/ok"))
					})
			}()

			time.Sleep(time.Second * 5) // to ensure request-1 always hits server before request-2

			// request-2: responses instantaneously
			err = f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 5, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})

			Expect(err).NotTo(HaveOccurred())       // request-1 should not block request-2 since maxconn = 2
			Expect(<-errChan).NotTo(HaveOccurred()) // check request-1

		})
	})

	Describe("With Pod MaxConnections (1) Specified", func() {
		BeforeEach(func() {
			meta, err := f.Ingress.CreateResourceWithBackendMaxConn(1)
			Expect(err).NotTo(HaveOccurred())

			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: meta.Name,
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
			}

			ing.Annotations[api.DefaultsTimeOut] = `{"connect": "300s", "server": "300s"}`
		})

		It("Should Allow 1 Connection Concurrently", func() {
			By("Getting HTTP endpoints")

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			errChan := make(chan error)
			go func() {
				// request-1: take 30s to response
				errChan <- f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 300, "", ing, eps, "GET",
					"/testpath/ok?delay=30s",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/testpath/ok"))
					})
			}()

			time.Sleep(time.Second * 5) // to ensure request-1 always hits server before request-2

			// request-2: responses instantaneously
			err = f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 5, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})

			// request-1 should block request-2 since maxconn = 1
			// request-2 should be timeout (sleep: 5s + client-timeout: 5s < request-1: 30s)
			Expect(err).To(HaveOccurred())
			Expect(<-errChan).NotTo(HaveOccurred()) // check request-1

		})
	})

	Describe("With Pod MaxConnections (2) Specified", func() {
		BeforeEach(func() {
			meta, err := f.Ingress.CreateResourceWithBackendMaxConn(2)
			Expect(err).NotTo(HaveOccurred())

			ing.Spec.Rules = []api.IngressRule{
				{
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
									Path: "/testpath",
									Backend: api.HTTPIngressBackend{
										IngressBackend: api.IngressBackend{
											ServiceName: meta.Name,
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				},
			}

			ing.Annotations[api.DefaultsTimeOut] = `{"connect": "300s", "server": "300s"}`
		})

		It("Should Allow 2 Connections Concurrently", func() {
			By("Getting HTTP endpoints")

			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			errChan := make(chan error)
			go func() {
				// request-1: take 30s to response
				errChan <- f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 300, "", ing, eps, "GET",
					"/testpath/ok?delay=30s",
					func(r *client.Response) bool {
						return Expect(r.Status).Should(Equal(http.StatusOK)) &&
							Expect(r.Method).Should(Equal("GET")) &&
							Expect(r.Path).Should(Equal("/testpath/ok"))
					})
			}()

			time.Sleep(time.Second * 5) // to ensure request-1 always hits server before request-2

			// request-2: responses instantaneously
			err = f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 5, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})

			Expect(err).NotTo(HaveOccurred())       // request-1 should not block request-2 since maxconn = 2
			Expect(<-errChan).NotTo(HaveOccurred()) // check request-1

		})
	})

	Describe("With Limit RPM", func() {
		BeforeEach(func() {
			ing.Annotations[api.LimitRPM] = "2"
		})

		It("Should Allow 2 Connections In one minute", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).To(HaveOccurred())

			klog.Warningln("Waiting 2 minute for timer to be reset")
			time.Sleep(time.Minute * 2)
			klog.Warningln("Request should response")
			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Limit Max Concurrent connection per ip", func() {
		BeforeEach(func() {
			ing.Annotations[api.LimitConnection] = "2"
			ing.Annotations[api.DefaultsTimeOut] = `{"connect": "300s", "server": "300s"}`
		})

		It("Should Allow 2 From IP Connections", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			errChan := make(chan error)
			for i := 1; i <= 2; i++ {
				go func() {
					err := f.Ingress.DoHTTPWithTimeout(framework.NoRetry, 150, "", ing, eps, "GET",
						"/testpath/ok?delay=60s",
						func(r *client.Response) bool {
							return Expect(r.Status).Should(Equal(http.StatusOK)) &&
								Expect(r.Method).Should(Equal("GET")) &&
								Expect(r.Path).Should(Equal("/testpath/ok"))
						},
					)
					errChan <- err
				}()
			}

			// Ensure this request must occurred after two long running request
			time.Sleep(time.Second * 10)
			err = f.Ingress.DoHTTP(framework.NoRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).To(HaveOccurred())

			Expect(<-errChan).NotTo(HaveOccurred())
			Expect(<-errChan).NotTo(HaveOccurred())

			klog.Warningln("Waiting 2 minute for timer to be reset")
			time.Sleep(time.Minute * 1)
			klog.Warningln("Request should response")
			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET",
				"/testpath/ok",
				func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("With Health Check Enabled", func() {
		Describe("Without Port Specified (Default Server's Port)", func() {
			BeforeEach(func() {
				svcAnnotation := map[string]string{
					api.CheckHealth: "true",
				}
				meta, err := f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
				Expect(err).NotTo(HaveOccurred())

				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: meta.Name,
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"option httpchk",
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

			It("Should Response HTTP", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(5, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With Correct Port but no Backend Rule", func() {
			BeforeEach(func() {
				svcAnnotation := map[string]string{
					api.CheckHealth:     "true",
					api.CheckHealthPort: "8080",
				}
				meta, err := f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
				Expect(err).NotTo(HaveOccurred())

				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName:  meta.Name,
												ServicePort:  intstr.FromInt(80),
												BackendRules: []string{},
											},
										},
									},
								},
							},
						},
					},
				}
			})

			It("Should Response HTTP", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(5, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With Correct Port and Backend Rule", func() {
			BeforeEach(func() {
				svcAnnotation := map[string]string{
					api.CheckHealth:     "true",
					api.CheckHealthPort: "8080",
				}
				meta, err := f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
				Expect(err).NotTo(HaveOccurred())

				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: meta.Name,
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"option httpchk GET /testpath/ok",
													"http-check expect rstring (testpath/ok)",
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

			It("Should Response HTTP", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTP(5, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath/ok"))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With Correct Port but Wrong Response Matching", func() {
			BeforeEach(func() {
				svcAnnotation := map[string]string{
					api.CheckHealth:     "true",
					api.CheckHealthPort: "8080",
				}
				meta, err := f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
				Expect(err).NotTo(HaveOccurred())

				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: meta.Name,
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"option httpchk GET /testpath/ok",
													"http-check expect rstring (wrongpath)",
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

			It("Should Not Response", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTPStatus(5, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("With Wrong Port", func() {
			BeforeEach(func() {
				svcAnnotation := map[string]string{
					api.CheckHealth:     "true",
					api.CheckHealthPort: "5050",
				}
				meta, err := f.Ingress.CreateResourceWithServiceAnnotation(svcAnnotation)
				Expect(err).NotTo(HaveOccurred())

				ing.Spec.Rules = []api.IngressRule{
					{
						IngressRuleValue: api.IngressRuleValue{
							HTTP: &api.HTTPIngressRuleValue{
								Paths: []api.HTTPIngressPath{
									{
										Path: "/testpath",
										Backend: api.HTTPIngressBackend{
											IngressBackend: api.IngressBackend{
												ServiceName: meta.Name,
												ServicePort: intstr.FromInt(80),
												BackendRules: []string{
													"option httpchk",
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

			It("Should Not Response", func() {
				By("Getting HTTP endpoints")
				eps, err := f.Ingress.GetHTTPEndpoints(ing)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(eps)).Should(BeNumerically(">=", 1))

				err = f.Ingress.DoHTTPStatus(5, ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
