package framework

import (
	vapi "github.com/appscode/voyager/apis/voyager"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyCRD() GomegaAsyncAssertion {
	return Eventually(func() error {
		crdClient := f.CRDClient.CustomResourceDefinitions()
		_, err := crdClient.Get(vapi.ResourceTypeIngress+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = crdClient.Get(vapi.ResourceTypeCertificate+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// TPR group registration has 10 sec delay inside Kubernetes api server. So, needs the extra check.
		_, err = f.VoyagerClient.VoyagerV1beta1().Ingresses(core.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		_, err = f.VoyagerClient.VoyagerV1beta1().Certificates(core.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		return nil
	})
}
