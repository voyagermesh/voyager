package certificate

import (
	"testing"

	etx "github.com/appscode/go/context"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	fakevoyager "github.com/appscode/voyager/client/fake"
	"github.com/appscode/voyager/pkg/config"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClient(t *testing.T) {
	controller, err := NewController(
		etx.Background(),
		fake.NewSimpleClientset(
			&apiv1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Data: map[string][]byte{
					api.ACMEUserEmail: []byte("test@test.com"),
					api.ACMEServerURL: []byte(LetsEncryptStagingURL),
				},
			},
		),
		fakevoyager.NewSimpleClientset(
			&api.Ingress{
				ObjectMeta: v1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
		).VoyagerV1beta1(),
		config.Options{},
		&api.Certificate{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: api.CertificateSpec{
				Domains: []string{"test.com"},
				ChallengeProvider: api.ChallengeProvider{
					HTTP: &api.HTTPChallengeProvider{
						Ingress: api.LocalTypedReference{
							APIVersion: "voyager.appscode.com/v1beta1",
							Kind:       "Ingress",
							Name:       "foo",
						},
					},
				},
				ACMEUserSecretName: "foo",
				Storage: api.CertificateStorage{
					Secret: &apiv1.LocalObjectReference{},
				},
			},
		},
	)
	assert.Nil(t, err)
	err = controller.getACMEClient()
	assert.Nil(t, err)
}
