package certificate

import (
	"testing"

	"github.com/appscode/voyager/apis/voyager/v1beta1"
	fakevoyager "github.com/appscode/voyager/client/fake"
	"github.com/appscode/voyager/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func TestClient(t *testing.T) {
	controller, err := NewController(
		fake.NewSimpleClientset(
			&apiv1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Data: map[string][]byte{
					v1beta1.ACMEUserEmail: []byte("test@test.com"),
					v1beta1.ACMEServerURL: []byte(LetsEncryptStagingURL),
				},
			},
		),
		fakevoyager.NewSimpleClientset(
			&v1beta1.Ingress{
				ObjectMeta: v1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
		).VoyagerV1beta1(),
		config.Options{},
		&v1beta1.Certificate{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: v1beta1.CertificateSpec{
				Domains: []string{"test.com"},
				ChallengeProvider: v1beta1.ChallengeProvider{
					HTTP: &v1beta1.HTTPChallengeProvider{
						Ingress: apiv1.ObjectReference{
							APIVersion: "voyager.appscode.com/v1beta1",
							Kind:       "Ingress",
							Name:       "foo",
							Namespace:  "bar",
						},
					},
				},
				ACMEUserSecretName: "foo",
				Storage: v1beta1.CertificateStorage{
					Secret: &v1beta1.SecretStore{},
				},
			},
		},
	)
	assert.Nil(t, err)
	err = controller.getACMEClient()
	assert.Nil(t, err)
}
