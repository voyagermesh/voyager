package framework

import (
	vapi "github.com/appscode/voyager/apis/voyager"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyCRD() GomegaAsyncAssertion {
	return Eventually(func() error {
		crdClient := f.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions()
		_, err := crdClient.Get(vapi.ResourceTypeIngress+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = crdClient.Get(vapi.ResourceTypeCertificate+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// TPR group registration has 10 sec delay inside Kubernetes api server. So, needs the extra check.
		_, err = f.V1beta1Client.Ingresses(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		_, err = f.V1beta1Client.Certificates(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		return nil
	})
}
