package operator

import (
	"github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/appscode/voyager/cmd/ssl-mounter/app/volume"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"os"
	"testing"
	"time"
)

func TestMount(t *testing.T) {
	fakeclient := fake.NewSimpleClientset(
		&apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "bar"},
			Data: map[string][]byte{
				apiv1.TLSPrivateKeyKey: []byte("world"),
				apiv1.TLSCertKey:       []byte("hello"),
			},
		},
	)

	location := os.TempDir() + "/tls-mount"
	assert.Nil(t, os.MkdirAll(location, os.ModePerm))

	m, err := NewIngressSecretMounter(
		fakeclient,
		nil,
		&v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "fix", Namespace: "bar"},
			Spec: v1beta1.IngressSpec{
				TLS: []v1beta1.IngressTLS{{SecretName: "user", Hosts: []string{"test.com"}}},
				Rules: []v1beta1.IngressRule{
					{Host: "test.com", IngressRuleValue: v1beta1.IngressRuleValue{HTTP: &v1beta1.HTTPIngressRuleValue{}}},
				},
			},
		},
		location,
		"",
		time.Minute,
	)
	if assert.Nil(t, err) {
		assert.Nil(t, m.Mount())
	}
}

func TestMountRemove(t *testing.T) {
	wr, err := volume.NewAtomicWriter(os.TempDir() + "/tls-mount/rm/")
	if assert.Nil(t, err) {
		payload := map[string]volume.FileProjection{
			"one": {Mode: 0777, Data: []byte("one")},
			"two": {Mode: 0777, Data: []byte("two")},
		}
		wr.Write(payload)

		delete(payload, "two")
		wr.Write(payload)
	}
}
