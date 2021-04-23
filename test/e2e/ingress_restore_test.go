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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Restore ingress offshoots", func() {
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

	It("Should restore configmap", func() {
		By("Deleting offshoot configmap")
		err := f.KubeClient.CoreV1().ConfigMaps(ing.Namespace).Delete(context.TODO(), ing.OffshootName(), metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Checking configmap restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	It("Should restore service", func() {
		By("Deleting offshoot service")
		err := f.KubeClient.CoreV1().Services(ing.Namespace).Delete(context.TODO(), ing.OffshootName(), metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Checking service restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})

	It("Should restore deployment", func() {
		By("Deleting haproxy deployment")
		err := f.KubeClient.AppsV1().Deployments(ing.Namespace).Delete(context.TODO(), ing.OffshootName(), metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Checking deployment restored")
		Expect(f.Ingress.IsExistsEventually(ing)).Should(BeTrue())
	})
})
