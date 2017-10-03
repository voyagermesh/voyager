package e2e

import (
	"net/http"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var _ = Describe("IngressWithBasicAuth", func() {
	var (
		f      *framework.Invocation
		ing    *api.Ingress
		secret *apiv1.Secret
	)

	BeforeEach(func() {
		f = root.Invoke()
		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
	})

	BeforeEach(func() {
		secret = &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Ingress.UniqueName(),
				Namespace: ing.GetNamespace(),
			},
			StringData: map[string]string{
				"auth": `foo::bar
				jane:E5BrlrQ5IXYK2`,

				"auth2": `auth2-foo::bar
				auth2-jane:E5BrlrQ5IXYK2`,
			},
		}
		_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
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
		if root.Config.Cleanup {
			f.Ingress.Delete(ing)
			f.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		}
	})

	Describe("Create", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}
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
			}
		})

		It("Should response HTTP weighted", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.NoRetry, ing, eps, "GET", "/testpath", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.MaxRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Service Auth", func() {
		BeforeEach(func() {
			meta, err := f.Ingress.CreateResourceWithServiceAuth(secret)
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
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.NoRetry, ing, eps, "GET", "/testpath", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic wrongPass",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).To(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	FDescribe("Both Global and Service Auth", func() {
		BeforeEach(func() {
			ing.Annotations = map[string]string{
				api.AuthType:   "basic",
				api.AuthRealm:  "Realm returned",
				api.AuthSecret: secret.Name,
			}

			secret2 := &apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.Ingress.UniqueName(),
					Namespace: ing.GetNamespace(),
				},
				StringData: map[string]string{
					"auth3": `auth3-foo::bar`,
				},
			}
			_, err := f.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret2)
			Expect(err).NotTo(HaveOccurred())

			meta, err := f.Ingress.CreateResourceWithServiceAuth(secret2)
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
		})

		It("Should response HTTP", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTPStatus(framework.NoRetry, ing, eps, "GET", "/testpath", func(r *testserverclient.Response) bool {
				return Expect(r.Status).Should(Equal(http.StatusUnauthorized))
			})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDMtZm9vOmJhcg==", // auth3-foo:bar
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).To(HaveOccurred()) // should error, since 'secret2' will be replaced by global 'secret'

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic Zm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic YXV0aDItZm9vOmJhcg==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTPWithHeader(
				framework.NoRetry,
				ing,
				eps,
				"GET",
				"/testpath",
				map[string]string{
					"Authorization": "Basic amFuZTpndWVzdA==",
				},
				func(r *testserverclient.Response) bool {
					return Expect(r.Status).Should(Equal(http.StatusOK)) &&
						Expect(r.Method).Should(Equal("GET")) &&
						Expect(r.Path).Should(Equal("/testpath"))
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
