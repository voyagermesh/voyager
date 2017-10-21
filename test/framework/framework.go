package framework

import (
	"sync"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/voyager/client/internalclientset/typed/voyager/internalversion"
	v1beta1client "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	. "github.com/onsi/gomega"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MaxRetry = 200
	NoRetry  = 1
)

type Framework struct {
	KubeConfig     *rest.Config
	KubeClient     kubernetes.Interface
	InternalClient internalversion.VoyagerInterface
	V1beta1Client  v1beta1client.VoyagerV1beta1Interface
	CRDClient      apiextensionsclient.Interface
	Config         E2EConfig
	namespace      string
	voyagerConfig  config.Options
	Mutex          sync.Mutex

	CertManager *CertManager
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

	cm, err := NewCertManager()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeConfig:     c,
		KubeClient:     kubernetes.NewForConfigOrDie(c),
		InternalClient: internalversion.NewForConfigOrDie(c),
		V1beta1Client:  v1beta1client.NewForConfigOrDie(c),
		CRDClient:      apiextensionsclient.NewForConfigOrDie(c),
		Config:         testConfigs,
		namespace:      testConfigs.TestNamespace,
		voyagerConfig: config.Options{
			CloudProvider:     testConfigs.CloudProviderName,
			HAProxyImage:      testConfigs.HAProxyImageName,
			IngressClass:      testConfigs.IngressClass,
			EnableRBAC:        testConfigs.RBACEnabled,
			OperatorNamespace: testConfigs.TestNamespace,
		},
		CertManager: cm,
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
