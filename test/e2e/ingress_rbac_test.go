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

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"
	"voyagermesh.dev/voyager/test/test-server/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/meta"
)

var _ = Describe("IngressWithRBACEnabled", func() {
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
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
		}
	})

	Describe("With RBAC", func() {
		BeforeEach(func() {
			if !meta.PossiblyInCluster() {
				Skip("RBAC can only be work in 'in-cluster' mode")
			}
		})

		It("Should test RBAC Support", func() {
			By("Getting HTTP endpoints")
			eps, err := f.Ingress.GetHTTPEndpoints(ing)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(eps)).Should(BeNumerically(">=", 1))

			_, err = f.KubeClient.CoreV1().ServiceAccounts(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.KubeClient.RbacV1beta1().Roles(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.KubeClient.RbacV1beta1().RoleBindings(ing.Namespace).Get(context.TODO(), ing.OffshootName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = f.Ingress.DoHTTP(framework.MaxRetry, "", ing, eps, "GET", "/testpath/ok", func(r *client.Response) bool {
				return Expect(r.Method).Should(Equal("GET")) &&
					Expect(r.Path).Should(Equal("/testpath/ok"))
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
