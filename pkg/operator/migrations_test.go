package operator

import (
	"log"
	"testing"

	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCertificateMigrations(t *testing.T) {
	op := &Operator{
		KubeClient: fake.NewSimpleClientset(),
	}

	for i, cert := range testInput {
		log.Println(i, cert.Spec.Domains)
		ok, err := op.MigrateCertificate(&cert)
		if assert.True(t, ok) && assert.Nil(t, err) {
			assert.Equal(t, cert, output[i])
		}
	}
}

var testInput []api.Certificate = []api.Certificate{
	{
		Spec: api.CertificateSpec{
			Provider:                     "http",
			HTTPProviderIngressReference: api.LocalTypedReference{Name: "test", APIVersion: "voyager.appscode.com/v1beta1"},
			Domains: []string{"http.appscode.com"},
		},
	},
}

var output []api.Certificate = []api.Certificate{
	{
		Spec: api.CertificateSpec{
			ChallengeProvider: api.ChallengeProvider{
				HTTP: &api.HTTPChallengeProvider{Ingress: api.LocalTypedReference{Name: "test", APIVersion: "voyager.appscode.com/v1beta1"}},
			},
			Domains:            []string{"http.appscode.com"},
			ACMEUserSecretName: "acme-",
			Storage:            api.CertificateStorage{Secret: &apiv1.LocalObjectReference{}},
		},
	},
}
