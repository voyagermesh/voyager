package testframework

import (
	vapi "github.com/appscode/voyager/api"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) EventuallyTPR() GomegaAsyncAssertion {
	return Eventually(func() error {
		_, err := f.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(vapi.ResourceNameIngress+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = f.KubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(vapi.ResourceNameCertificate+"."+vapi.GroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// TPR group registration has 10 sec delay inside Kubernetes api server. So, needs the extra check.
		_, err = f.VoyagerClient.Ingresses(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		_, err = f.VoyagerClient.Certificates(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		return nil
	})
}
