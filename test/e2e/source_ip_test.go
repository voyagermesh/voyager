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
	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/test/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
)

var _ = Describe("With health-check-nodeport annotation", func() {
	var (
		f   *framework.Invocation
		ing *api.Ingress
	)

	BeforeEach(func() {
		f = root.Invoke()

		switch options.CloudProvider {
		case "gce", api.ProviderGKE, "azure", "acs", "aks", "metallb":
		default:
			Skip(options.CloudProvider + " do not support this")
		}

		ing = f.Ingress.GetSkeleton()
		f.Ingress.SetSkeletonRule(ing)
		ing.Annotations = map[string]string{
			api.KeepSourceIP:        "true",
			api.HealthCheckNodeport: "32456",
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
			Expect(f.Ingress.Delete(ing)).NotTo(HaveOccurred())
		}
	})

	Describe("Create", func() {
		It("Should set specified port in service's healthCheckNodePort", func() {
			Eventually(func() bool {
				svc, err := f.Ingress.GetOffShootService(ing)
				if err != nil {
					return false
				}
				if svc.Spec.ExternalTrafficPolicy != core.ServiceExternalTrafficPolicyTypeLocal || svc.Spec.HealthCheckNodePort != 32456 {
					return false
				}
				return true
			}, "5m", "10s").Should(BeTrue())
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			ing.Annotations[api.HealthCheckNodeport] = "32466"
		})
		It("Should update healthCheckNodePort", func() {
			Eventually(func() bool {
				svc, err := f.Ingress.GetOffShootService(ing)
				if err != nil {
					return false
				}
				if svc.Spec.ExternalTrafficPolicy != core.ServiceExternalTrafficPolicyTypeLocal || svc.Spec.HealthCheckNodePort != 32466 {
					return false
				}
				return true
			}, "3m", "10s").Should(BeTrue())
		})
	})
})
