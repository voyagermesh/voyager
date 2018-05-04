package e2e

import (
	"net"
	"net/http"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("IngressWithWildCardDomain", func() {
	var (
		f      *framework.Invocation
		ing    *api.Ingress
		secret *core.Secret
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
			f.Ingress.Delete(ing)
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: "*.appscode.test",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
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

		It("Should response HTTP from WildCard Host", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.MaxRetry, "test-1.appscode.test", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath")) &&
					Expect(r.Host).Should(Equal("test-1.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, "test-2.appscode.test", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath")) &&
					Expect(r.Host).Should(Equal("test-2.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, "anything.appscode.test", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath")) &&
					Expect(r.Host).Should(Equal("anything.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, "everything.anything.appscode.test", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath")) &&
					Expect(r.Host).Should(Equal("everything.anything.appscode.test"))
			})
			Expect(err).NotTo(HaveOccurred())

			// Fail
			err = f.Ingress.DoHTTPStatus(framework.NoRetry, ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPStatusWithHost(framework.NoRetry, "appscode.com", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateWithTLS", func() {
		BeforeEach(func() {
			var err error
			secret, err = f.Ingress.CreateTLSSecretForHost(f.UniqueName(), []string{framework.TestDomain})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if options.Cleanup {
				f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
			}
		})

		BeforeEach(func() {
			if options.CloudProvider == "minikube" {
				ing.Annotations[api.LBType] = api.LBTypeHostPort
			}
			ing.Annotations[api.SSLRedirect] = "false"
			ing.Spec.TLS = []api.IngressTLS{
				{
					Ref: &api.LocalTypedReference{
						Kind: "Secret",
						Name: secret.Name,
					},
					Hosts: []string{"*.appscode.test"},
				},
			}
			ing.Spec.Rules = []api.IngressRule{
				{
					Host: "*.appscode.test",
					IngressRuleValue: api.IngressRuleValue{
						HTTP: &api.HTTPIngressRuleValue{
							Paths: []api.HTTPIngressPath{
								{
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

		It("Should response HTTP from WildCard HTTPS", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			resolved := false
			log.Warningln("Domain 'http.appscode.test' must resolve to ips in", eps)

			// Checking for the domain is pointing to the ips found in the endpoints
			// The IPs and domain must be in /etc/hosts file
			ips, err := net.LookupHost(framework.TestDomain)
			if err != nil || len(ips) == 0 {
				Skip("Domain 'http.appscode.test' do not have endpoints")
			}

		Outer:
			for _, ep := range eps {
				vep := strings.TrimLeft(ep[:strings.LastIndex(ep, ":")], "http://")
				for _, ip := range ips {
					if vep == ip {
						resolved = true
						break Outer
					}
				}
			}

			if !resolved {
				Skip("Domain 'http.appscode.test' did not point to endpoints")
			}

			err = f.Ingress.DoHTTPs(framework.MaxRetry, framework.TestDomain, "", ing, []string{"https://http.appscode.test"}, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusOK)) &&
					Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath")) &&
					Expect(r.Host).Should(Equal(framework.TestDomain))
			})
			Expect(err).NotTo(HaveOccurred())

			// Fail
			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
			})
			Expect(err).To(HaveOccurred())

			err = f.Ingress.DoHTTPsStatus(framework.NoRetry, "appscode.com", ing, eps, "GET", "/testpath", func(r *client.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusServiceUnavailable))
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
