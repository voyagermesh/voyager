package testframework

import (
	"github.com/appscode/go/crypto/rand"
	voyagerclient "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	. "github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	maxRetryCount = 50
)

type Framework struct {
	KubeClient    clientset.Interface
	VoyagerClient voyagerclient.ExtensionInterface
	Config        E2EConfig
	namespace     string
	voyagerConfig config.Options
}

type Invocation struct {
	*rootInvocation
	Ingress *ingressInvocation
}

type rootInvocation struct {
	*Framework
	app string
}

type ingressInvocation struct {
	*rootInvocation
}

func New() *Framework {
	testConfigs.validate()

	c, err := clientcmd.BuildConfigFromFlags(testConfigs.Master, testConfigs.KubeConfig)
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:    clientset.NewForConfigOrDie(c),
		VoyagerClient: voyagerclient.NewForConfigOrDie(c),
		Config:        testConfigs,
		namespace:     testConfigs.TestNamespace,
		voyagerConfig: config.Options{
			CloudProvider:     testConfigs.CloudProviderName,
			HAProxyImage:      testConfigs.HAProxyImageName,
			IngressClass:      testConfigs.IngressClass,
			EnableRBAC:        testConfigs.RBACEnabled,
			OperatorNamespace: testConfigs.TestNamespace,
		},
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
		Ingress:        &ingressInvocation{r},
	}
}
