package operator

import (
	"os"
	"testing"

	api "github.com/appscode/voyager/apis/voyager"
	"github.com/stretchr/testify/assert"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestEnsureCustomResourceDefinitions(t *testing.T) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		t.SkipNow()
	}

	crdClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		t.SkipNow()
	}

	_, err = crdClient.Discovery().ServerVersion()
	if err != nil {
		t.Skip("No Server found, skipping")
	}

	op := Operator{CRDClient: crdClient, KubeClient: kubernetes.NewForConfigOrDie(config)}
	if assert.Nil(t, op.ensureCustomResourceDefinitions()) {
		rs, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("ingresses.voyager.appscode.com", v1.GetOptions{})
		if assert.Nil(t, err) {
			assert.Equal(t, rs.Name, "ingresses.voyager.appscode.com")
		}
	}

	voyagerClient := clientset.NewForConfigOrDie(config)
	_, err = voyagerClient.Ingresses("default").Create(&api.Ingress{
		ObjectMeta: v1.ObjectMeta{Name: "test-ingress", Namespace: "default"},
		Spec: api.IngressSpec{Rules: []api.IngressRule{{IngressRuleValue: api.IngressRuleValue{
			HTTP: &api.HTTPIngressRuleValue{
				Paths: []api.HTTPIngressPath{{Backend: api.HTTPIngressBackend{
					IngressBackend: api.IngressBackend{ServiceName: "foo", ServicePort: intstr.FromInt(80)},
				}}},
			},
		}}}},
	})
	if assert.Nil(t, err) {
		v, err := voyagerClient.Ingresses("default").Get("test-ingress")
		assert.Nil(t, err)
		assert.Equal(t, v.Name, "test-ingress")

		defer voyagerClient.Ingresses("default").Delete("test-ingress")
	}

	_, err = voyagerClient.Certificates("default").Create(&api.Certificate{
		ObjectMeta: v1.ObjectMeta{Name: "test-cert", Namespace: "default"},
	})
	if assert.Nil(t, err) {
		v, err := voyagerClient.Certificates("default").Get("test-cert")
		assert.Nil(t, err)
		assert.Equal(t, v.Name, "test-cert")

		defer voyagerClient.Certificates("default").Delete("test-cert")
	}

	crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete("ingresses.voyager.appscode.com", &v1.DeleteOptions{})
	crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete("certificates.voyager.appscode.com", &v1.DeleteOptions{})
}
