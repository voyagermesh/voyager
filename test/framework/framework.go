package framework

import (
	"sync"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/kutil/tools/certstore"
	cs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MaxRetry   = 200
	NoRetry    = 1
	TestDomain = "http.appscode.test"
)

type Framework struct {
	KubeConfig    *rest.Config
	KubeClient    kubernetes.Interface
	VoyagerClient cs.VoyagerV1beta1Interface
	CRDClient     kext_cs.ApiextensionsV1beta1Interface
	Config        E2EConfig
	namespace     string
	voyagerConfig config.Options
	Mutex         sync.Mutex

	CertStore *certstore.CertStore
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

func New() *Framework {
	testConfigs.validate()

	c, err := clientcmd.BuildConfigFromFlags(testConfigs.Master, testConfigs.KubeConfig)
	Expect(err).NotTo(HaveOccurred())

	cm, err := certstore.NewCertStore(afero.NewMemMapFs(), "/pki")
	Expect(err).NotTo(HaveOccurred())

	err = cm.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeConfig:    c,
		KubeClient:    kubernetes.NewForConfigOrDie(c),
		VoyagerClient: cs.NewForConfigOrDie(c),
		CRDClient:     kext_cs.NewForConfigOrDie(c),
		Config:        testConfigs,
		namespace:     testConfigs.TestNamespace,
		voyagerConfig: config.Options{
			CloudProvider:     testConfigs.CloudProviderName,
			HAProxyImage:      testConfigs.HAProxyImageName,
			IngressClass:      testConfigs.IngressClass,
			EnableRBAC:        testConfigs.RBACEnabled,
			OperatorNamespace: testConfigs.TestNamespace,
		},
		CertStore: cm,
	}
}

func (f *Framework) VoyagerConfig() config.Options {
	return f.voyagerConfig
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
