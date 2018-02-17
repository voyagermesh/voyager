package operator

import (
	"os"
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/stretchr/testify/assert"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
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

	crdClient, err := kext_cs.NewForConfig(config)
	if err != nil {
		t.SkipNow()
	}

	op := Operator{CRDClient: crdClient, KubeClient: kubernetes.NewForConfigOrDie(config)}
	if assert.Nil(t, op.Setup()) {
		rs, err := crdClient.CustomResourceDefinitions().Get("ingresses.voyager.appscode.com", v1.GetOptions{})
		if assert.Nil(t, err) {
			assert.Equal(t, rs.Name, "ingresses.voyager.appscode.com")
		}
	}

	voyagerClient := cs.NewForConfigOrDie(config)
	_, err = voyagerClient.VoyagerV1beta1().Ingresses("default").Create(&api.Ingress{
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
		v, err := voyagerClient.VoyagerV1beta1().Ingresses("default").Get("test-ingress", v1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, v.Name, "test-ingress")

		defer voyagerClient.VoyagerV1beta1().Ingresses("default").Delete("test-ingress", &v1.DeleteOptions{})
	}

	_, err = voyagerClient.VoyagerV1beta1().Certificates("default").Create(&api.Certificate{
		ObjectMeta: v1.ObjectMeta{Name: "test-cert", Namespace: "default"},
	})
	if assert.Nil(t, err) {
		v, err := voyagerClient.VoyagerV1beta1().Certificates("default").Get("test-cert", v1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, v.Name, "test-cert")

		defer voyagerClient.VoyagerV1beta1().Certificates("default").Delete("test-cert", &v1.DeleteOptions{})
	}

	crdClient.CustomResourceDefinitions().Delete("ingresses.voyager.appscode.com", &v1.DeleteOptions{})
	crdClient.CustomResourceDefinitions().Delete("certificates.voyager.appscode.com", &v1.DeleteOptions{})
}
