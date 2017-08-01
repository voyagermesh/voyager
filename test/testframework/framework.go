package testframework

import (
	"github.com/appscode/go/crypto/rand"
	voyagerclient "github.com/appscode/voyager/client/clientset"
	. "github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Framework struct {
	KubeClient    clientset.Interface
	VoyagerClient voyagerclient.ExtensionInterface
	Config        E2EConfig
	namespace     string
}

type Invocation struct {
	*Framework
	app string
}

func New() *Framework {
	config.validate()

	c, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:    clientset.NewForConfigOrDie(c),
		VoyagerClient: voyagerclient.NewForConfigOrDie(c),
		Config:        config,
		namespace:     config.TestNamespace,
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("voyager-e2e"),
	}
}
