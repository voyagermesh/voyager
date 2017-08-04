package e2e

import (
	"time"

	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/framework"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
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
		_, err := f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Create(ext)
		Expect(err).NotTo(HaveOccurred())

		f.Ingress.EventuallyStarted(ing).Should(BeTrue())

		By("Checking generated resource")
		Expect(f.Ingress.IsTargetCreated(ing)).Should(BeTrue())
	})

	AfterEach(func() {
		if root.Config.Cleanup {
			f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Delete(ext.Name, &metav1.DeleteOptions{})
		}
	})

	var (
		shouldCreateServiceEntry = func() {
			By("Checking StatusIP for provider" + f.Config.CloudProviderName)
			if f.Config.CloudProviderName == "minikube" {
				Skip("Minikube do not support this")
			}
			// Check Status for ingress
			baseIngress, err := f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Get(ext.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			svc, err := f.Ingress.GetOffShootService(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(baseIngress.Status.LoadBalancer.Ingress)).Should(Equal(len(svc.Status.LoadBalancer.Ingress)))
			Expect(baseIngress.Status.LoadBalancer.Ingress[0]).Should(Equal(svc.Status.LoadBalancer.Ingress[0]))
		}

		shouldResponseHTTP = func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			err = f.Ingress.DoHTTP(framework.MaxRetry, ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())
		}

		shouldDeleteResource = func() {
			By("Deleting Ingress resource")
			err := f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Delete(ext.Name, &metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			Eventually(f.Ingress.Controller(ing).IsExists, "5m", "10s").Should(BeFalse())
		}

		shouldUpdateLoadbalancer = func() {
			By("Updating Ingress resource")
			uing, err := f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Get(ext.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			uing.Spec.Rules[0].HTTP.Paths[0].Path = "/newTestPath"
			_, err = f.KubeClient.ExtensionsV1beta1().Ingresses(ext.Namespace).Update(uing)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting some time for update to be applied")
			time.Sleep(time.Second * 5)

			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			By("Calling new HTTP path")
			err = f.Ingress.DoHTTP(framework.MaxRetry, ing, eps, "GET", "/newTestPath/ok", func(r *testserverclient.Response) bool {
				return Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/newTestPath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking old path")
			err = f.Ingress.DoHTTP(framework.NoRetry, ing, eps, "GET", "/testpath/ok", func(r *testserverclient.Response) bool {
				return true
			})
			Expect(err).To(HaveOccurred())
		}
	)

	Describe("Create", func() {
		It("Should create Loadbalancer entry", shouldCreateServiceEntry)
		It("Should response HTTP", shouldResponseHTTP)
	})

	Describe("Delete", func() {
		It("Should delete Ingress resource", shouldDeleteResource)
	})

	Describe("Update", func() {
		It("Should update Loadbalancer", shouldUpdateLoadbalancer)
	})
})
