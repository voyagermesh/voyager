package framework

import (
	"sync"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/kutil/tools/certstore"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/appscode/voyager/pkg/operator"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
)

const (
	MaxRetry   = 10
	NoRetry    = 1
	TestDomain = "http.appscode.test"
)

type Framework struct {
	KubeClient    kubernetes.Interface
	VoyagerClient cs.Interface
	CRDClient     kext_cs.ApiextensionsV1beta1Interface

	Operator      *operator.Operator
	CertStore     *certstore.CertStore
	TestNamespace string
	Cleanup       bool

	Lock sync.Mutex
}

type Invocation struct {
	*rootInvocation
	Ingress     *ingressInvocation
	Certificate *certificateInvocation
}

type rootInvocation struct {
	*Framework
	app string
}

type ingressInvocation struct {
	*rootInvocation
}

type certificateInvocation struct {
	*rootInvocation
}

func New(cfg *operator.OperatorConfig, testNamespace string, cleanup bool) *Framework {
	op, err := cfg.New()
	Expect(err).NotTo(HaveOccurred())

	cm, err := certstore.NewCertStore(afero.NewMemMapFs(), "/pki")
	Expect(err).NotTo(HaveOccurred())

	err = cm.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:    cfg.KubeClient,
		VoyagerClient: cfg.VoyagerClient,
		CRDClient:     cfg.CRDClient,
		Operator:      op,
		CertStore:     cm,
		TestNamespace: testNamespace,
		Cleanup:       cleanup,
	}
}

func (f *Framework) Invoke() *Invocation {
	r := &rootInvocation{
		Framework: f,
		app:       rand.WithUniqSuffix("voyager-e2e"),
	}
	return &Invocation{
		rootInvocation: r,
		Ingress:        &ingressInvocation{rootInvocation: r},
		Certificate:    &certificateInvocation{rootInvocation: r},
	}
}

func (f *rootInvocation) App() string {
	return f.app
}
